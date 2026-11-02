# Badges

The README badges, generated as local SVG files rather than served by
shields.io, so the README depends on nothing outside the repository.

Three badges are the exception: the latest release, the CI status and the
stars. They change without any commit to this folder, so they are served by
shields.io and GitHub.

## Regenerate

From the repository root:

```bash
npm i simple-icons
python3 .github/badges/generate.py
```

Needs Python 3 with Pillow and the DejaVu Sans font. Edit the `BADGES` table at
the bottom of `generate.py` to change a label, a message or a colour.
