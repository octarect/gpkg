# gpkg

[![CI](https://github.com/octarect/gpkg/actions/workflows/ci.yaml/badge.svg)](https://github.com/octarect/gpkg/actions/workflows/ci.yaml)

A package manager written in Go. (Currently in development 🔨)

## Getting started

### Installation

### Add a package

```toml
[[packages]]
from = "ghr"
repo = "junegunn/fzf"
```

### Load packages

Installed plugins can be loaded using `load`.

```bash
eval $(gpkg load)
```

## License

MIT License (© 2023 Ryota Kota)
