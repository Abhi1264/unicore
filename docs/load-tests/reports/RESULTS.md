# Load-test report template

## Environment
- Date:
- Hardware / cloud:
- Redis: enabled / disabled
- Seed: N students, M published results

## `/results` (k6)

| Mode | VUs | Duration | p50 | p95 | error rate |
|------|-----|----------|-----|-----|------------|
| Uncached (Redis off) | | | | | |
| Cached (Redis on) | | | | | |

## Course registration seat-cap race

| Cap | Concurrent attempts | Final enrollments | Oversell? |
|-----|---------------------|-------------------|-----------|
| 5 | 20 (Go race test) | 5 | no |
| | | | |

## Notes

Capture Grafana screenshots under `docs/load-tests/reports/` (gitignored binaries OK; commit markdown + small PNGs).
