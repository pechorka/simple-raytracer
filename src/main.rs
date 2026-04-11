use clap::Parser;

use macroquad::prelude::*;

#[derive(Parser)]
struct Config {
    #[arg(short = 'W', long, default_value_t = 1000)]
    w: usize,
    #[arg(short = 'H', long, default_value_t = 1000)]
    h: usize,
}

#[macroquad::main("BasicShapes")]
async fn main() {
    let cfg = Config::parse();
    let mut image = Image::gen_image_color(cfg.w as u16, cfg.h as u16, YELLOW);
    let texture = Texture2D::from_image(&image);

    loop {
        if is_key_pressed(KeyCode::Escape) {
            return;
        }

        let render_ctx = RenderCtx {
            w: cfg.w,
            h: cfg.h,
            elapsed: get_time() as f32,
        };
        let mut buf = image.get_image_data_mut();
        render_sphere(&render_ctx, &mut buf);

        texture.update(&image);
        draw_texture(&texture, 0.0, 0.0, WHITE);

        next_frame().await
    }
}

struct RenderCtx {
    w: usize,
    h: usize,
    elapsed: f32,
}

fn render_sphere(ctx: &RenderCtx, buf: &mut [[u8; 4]]) {
    let w = ctx.w;
    let h = ctx.h;

    let origin = Vec3::ZERO;

    let sc = vec3(0.0, 0.0, 1.0);
    let sr: f32 = 0.3;
    let el = ctx.elapsed;
    let lp = vec3(sc.x + sr * el.cos(), sc.y + sr * el.cos(), 0.3);

    let aspect = w as f32 / h as f32;

    for y in 0..h {
        for x in 0..w {
            let i = y * w + x;
            buf[i] = [0, 0, 0, 255];

            let u = (x as f32 / w as f32) * 2.0 - 1.0;
            let v = (y as f32 / h as f32) * 2.0 - 1.0;

            let point = vec3(u * aspect, -v, 1.0);
            let dir = point - origin;

            let a = dir.dot(dir);
            let b = 2.0 * dir.dot(origin - sc);
            let c = (origin - sc).dot(origin - sc) - sr * sr;

            let d = b * b - 4.0 * a * c;
            if d <= 0.0 {
                continue;
            }

            let t1 = (-b - d.sqrt()) / (2.0 * a);
            let t2 = (-b + d.sqrt()) / (2.0 * a);
            if t1 <= 0.0 && t2 <= 0.0 {
                continue;
            }

            let t = if t1 <= 0.0 || t1 > t2 { t2 } else { t1 };

            let hit_point = origin + dir * t;
            let surface_normal = (hit_point - sc).normalize();

            let l = (lp - hit_point).normalize();

            let red = (200.0 * surface_normal.dot(l).max(0.0)).clamp(0.0, 255.0) as u8;
            buf[i] = [red, 0, 0, 255];
        }
    }
}
