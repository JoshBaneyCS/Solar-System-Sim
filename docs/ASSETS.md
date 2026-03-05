# Assets

## Structure

```
assets/
  textures/
    {planet}/albedo.jpg    — per-body albedo textures (JPEG or PNG)
    skybox/milky_way.jpg   — background starfield
  models/
    earth.glb              — Earth 3D model (glTF 2.0)
  meshes/
    sphere_32.glb          — generated UV sphere (32 segments)
    sphere_64.glb          — generated UV sphere (64 segments)
  CREDITS.md               — licensing and attribution
```

## How Textures Work

Textures are loaded at startup into a GPU texture array (one layer per body). The SDF circle fragment shader computes spherical UV coordinates from the disk position and samples the texture, giving the appearance of a textured sphere in orthographic view. The ray tracer also supports texture sampling at sphere intersection points.

Body ordering in the texture array: mercury(0), venus(1), earth(2), mars(3), jupiter(4), saturn(5), uranus(6), neptune(7), sun(8).

If textures are missing, the renderer falls back to solid colors automatically.

## Setup

Run `make assets-setup` to copy source textures from `space-object-textures/` into the canonical `assets/` layout. This is required before running the GPU renderer with textures.

## Tools

- `cmd/meshgen` — generates UV sphere meshes as `.glb` files (`make meshgen`)
- `cmd/validate-assets` — validates asset directory structure and file integrity (`make validate-assets`)

## Adding New Textures

1. Place the albedo texture in `assets/textures/{body_name}/albedo.jpg` (or `.png`)
2. The body name must match one of: mercury, venus, earth, mars, jupiter, saturn, uranus, neptune, sun
3. Textures are automatically resized to 2048x1024 on load
4. Update `assets/CREDITS.md` with source and license information
5. Run `make validate-assets` to verify

## Notes

- All textures are resized to a uniform 2048x1024 resolution for the GPU texture array
- Equirectangular projection is expected (standard for planet textures)
- Licensing documented in `assets/CREDITS.md`
