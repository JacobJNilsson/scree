# Corpus results

Every row is one public module at one pinned version, audited with no
configuration. The index is 0 to 100 and lower is better. The two points columns
hold the points that each dimension contributed to the index. Production counts
cover the measured production set only. A run of make corpus rewrites this file.

| module | version | index | complexity-erosion points | duplication points | production code lines | production functions | eroded functions | clone groups | completeness |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| github.com/mattn/go-isatty | v0.0.20 | 17 | 17 | 0 | 142 | 15 | 1 | 0 | complete |
| github.com/pkg/errors | v0.9.1 | 0 | 0 | 0 | 277 | 32 | 0 | 0 | complete |
| github.com/fatih/color | v1.17.0 | 17 | 0 | 17 | 392 | 83 | 0 | 6 | complete |
| github.com/google/uuid | v1.6.0 | 27 | 19 | 8 | 805 | 72 | 3 | 2 | complete |
| github.com/go-chi/chi/v5 | v5.1.0 | 25 | 16 | 9 | 2655 | 234 | 5 | 7 | complete |
| github.com/gorilla/websocket | v1.5.3 | 37 | 31 | 6 | 2938 | 165 | 10 | 3 | complete |
| github.com/stretchr/testify | v1.9.0 | 48 | 28 | 20 | 3180 | 205 | 10 | 13 | complete |
| go.uber.org/zap | v1.27.0 | 40 | 20 | 20 | 4897 | 663 | 7 | 25 | complete |
| github.com/rs/zerolog | v1.33.0 | 59 | 28 | 31 | 5611 | 568 | 16 | 45 | complete |
| gopkg.in/yaml.v3 | v3.0.1 | 67 | 48 | 19 | 7710 | 302 | 61 | 29 | complete |
| github.com/prometheus/client_golang | v1.19.1 | 49 | 27 | 22 | 9330 | 657 | 23 | 34 | complete |
| github.com/valyala/fasthttp | v1.55.0 | 59 | 36 | 23 | 14392 | 1172 | 56 | 55 | complete |
| k8s.io/apimachinery | v0.31.0 | 56 | 34 | 22 | 29518 | 2435 | 91 | 114 | complete |
| google.golang.org/grpc | v1.65.0 | 54 | 35 | 19 | 50191 | 3644 | 137 | 127 | complete |
| github.com/prometheus/prometheus | v0.54.0 | 68 | 42 | 26 | 75973 | 4660 | 285 | 324 | complete |

Over the 7 modules above 5000 production lines, the Spearman rank correlation of the index is +0.05 with production code lines and +0.83 with debt per thousand production lines. Debt is eroded functions plus clone groups.
