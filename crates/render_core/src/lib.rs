#![allow(
    clippy::missing_safety_doc,
    clippy::needless_range_loop,
    clippy::collapsible_if,
    clippy::too_many_arguments,
    clippy::manual_div_ceil,
    clippy::manual_ok_err,
    clippy::empty_line_after_doc_comments,
    dead_code
)]

pub mod camera;
pub mod ffi;
pub mod hardware;
pub mod pipeline;
pub mod raytracer;
pub mod renderer;
pub mod shapes;
pub mod spacetime;
pub mod textures;
