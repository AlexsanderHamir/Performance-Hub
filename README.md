# Performance Hub

Automations for **performance** (and other areas), written in **multiple languages**. Each automation lives in one folder and can be run on its own.

## Layout

| Language | Categories |
|----------|------------|
| **Go** | [performance](go/performance/) |
| **Python** | [performance](python/performance/) |

Add new automations under the right language and category (e.g. `go/performance/my-tool/`).

## Quick start

- **Go**  
  From the `go/` directory: `cd go && go run ./...` (or run a specific automation under `go/<category>/<name>/`).
- **Python**  
  Use a venv and install deps from the automation folder:  
  `cd python/performance/<name> && pip install -r requirements.txt && python main.py` (or the script you add).

## Design

See **[docs/DESIGN.md](docs/DESIGN.md)** for the full design: folder rules, how to add automations and new languages, and how dependencies are managed.

## Adding an automation

1. Choose language (e.g. `go/`) and category (e.g. `performance/`).
2. Create a folder: `go/performance/<automation-name>/`.
3. Add code, a README (what it does, how to run it), and dependency files.
4. Optionally add a one-line link in this README under the right language/category.
