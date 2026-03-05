# Assets

> This doc is expanded by the `assets-pipeline` agent.

## Structure
- `assets/models/` — `.glb` models (Earth, optional ships)
- `assets/textures/{body}/` — albedo/normal/roughness/emissive maps
- `assets/backgrounds/` — starfields/HDRI

## Planet Textures + Sphere Meshes
If only textures exist, generate a sphere mesh and apply UV mapping.

## Tools
- `tools/meshgen` — generate sphere meshes
- `tools/validate_assets` — verify presence and sizes

## Notes
- Document licensing for every asset in `docs/CREDITS.md`.
