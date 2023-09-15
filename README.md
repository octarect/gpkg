# gpkg

Super great package manager.

## Getting started

### Installation

### Add a package

```toml
[[packages]]
from = "ghr"
name = "junegunn/fzf"
```

### Load packages

Installed plugins can be loaded using `load`.

```bash
eval $(gpkg load)
```

## License

MIT License (© 2023 Ryota Kota)
