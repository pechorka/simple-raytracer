use clap::Parser;
use macroquad::prelude::*;

#[derive(clap::Parser)]
struct Config {
    #[arg(short = 'W', long, default_value_t = 1000)]
    w: usize,
    #[arg(short = 'H', long, default_value_t = 1000)]
    h: usize,
}

#[macroquad::main("BasicShapes")]
async fn main() {
    let cfg = Config::try_parse().unwrap_or_else(|e| e.exit());
    let mut image = Image::gen_image_color(cfg.w as u16, cfg.h as u16, BLACK);
    let texture = Texture2D::from_image(&image);

    let mut objs = Objects { spheres: vec![] };

    loop {
        if is_key_pressed(KeyCode::Escape) {
            return;
        }

        if is_mouse_button_pressed(MouseButton::Left) {
            let mouse = mouse_position_local();
            objs.spheres.push(Sphere {
                center: vec3(mouse.x, -mouse.y, 1.0),
                radius: 0.3,
            });
        }

        let render_ctx = RenderCtx {
            w: cfg.w,
            h: cfg.h,
            elapsed: get_time() as f32,
        };
        let mut buf = image.get_image_data_mut();
        render(&render_ctx, &mut buf, &objs);

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

struct Objects {
    spheres: Vec<Sphere>,
}

struct Sphere {
    center: Vec3,
    radius: f32,
}

fn render(ctx: &RenderCtx, buf: &mut [[u8; 4]], objs: &Objects) {
    let w = ctx.w;
    let h = ctx.h;

    let origin = Vec3::ZERO;

    let el = ctx.elapsed;
    let lp = vec3(el.sin() * 1.0, el.cos() * -1.0, 1.0);

    let aspect = w as f32 / h as f32;

    for y in 0..h {
        for x in 0..w {
            let i = y * w + x;
            buf[i] = [0, 0, 0, 255];

            let u = (x as f32 / w as f32) * 2.0 - 1.0;
            let v = (y as f32 / h as f32) * 2.0 - 1.0;

            let point = vec3(u * aspect, -v, 1.0);
            let dir = point - origin;

            for s in objs.spheres.iter() {
                let sc = s.center;
                let sr = s.radius;

                let a = dir.dot(dir);
                let b = 2.0 * dir.dot(origin - sc);
                let c = (origin - sc).dot(origin - sc) - sr * sr;

                let d = b * b - 4.0 * a * c;

                if d > 0.0 {
                    let t = (-b - d.sqrt()) / (2.0 * a);
                    if t > 0.0 {
                        let hit_point = origin + dir * t;
                        let surface_normal = (hit_point - sc).normalize();

                        let l = (lp - hit_point).normalize();

                        let red = (200.0 * surface_normal.dot(l).max(0.0)).clamp(0.0, 255.0) as u8;
                        buf[i] = [red, 0, 0, 255];
                    }
                }
            }
        }
    }
}
