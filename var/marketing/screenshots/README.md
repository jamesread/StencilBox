# Marketing screenshots

The screenshots used in the project README, captured from a throwaway
StencilBox running the fictional config in [`seed/`](seed/).

```bash
make screenshots                      # from the repo root
make -C var/marketing/screenshots     # or from anywhere
```

This builds the service and web UI, starts StencilBox on `127.0.0.1:18480`
with a copy of `seed/` in `tmp/`, waits for startup builds of the example
sites, captures every section of `screenshots.ini` with `repo-helper
screenshot`, regenerates `screenshot-manifest.yaml`, copies selected images
into the Antora docs images directory, and then stops StencilBox, even if a
step fails.

## Requirements

- Go and Node.js, to build the service and frontend
- [`repo-helper`](https://github.com/jamesread/repo-common), with Chrome and a
  matching `chromedriver`
- `curl` and `jq`

No containers are needed: StencilBox stores users in a SQLite file next to
`config.yaml`.

## Files

| File | Purpose |
|------|---------|
| `seed/config.yaml`, `seed/buildconfigs/` | Fictional config and YAML data. Sidebar icons live under `repos/demo-icons/` (tiny local PNGs) so the build does not clone external icon repos. |
| `seed.sh` | Logs in as the default admin and waits for startup builds. |
| `screenshots.ini` | One section per image, plus the `title`/`description`/`theme` used for the manifest. |
| `scripts/` | Selenium setup scripts. `light.py`/`dark.py` pin the PicoCrank theme because repo-helper shares one browser session across sections. |
| `generate-manifest.py` | Writes `screenshot-manifest.yaml`. |
| `screenshot-manifest.yaml` | Generated index of the screenshots for indexers. Do not edit by hand. |

To add or change a screenshot, edit `screenshots.ini` and run `make`. If only
a title or description changed, `make manifest` is enough.

To browse the demo instance yourself, run `make build up seed`, open
<http://127.0.0.1:18480/webui/>, and run `make down` when finished.
