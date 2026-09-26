# wend.press

Global trajectory intelligence for the world's news.
Trajectories, not predictions. Auditable, not opaque.

## Layout

    cmd/wend            entrypoint
    internal/config     env + config loading
    internal/domain     canonical types (Source, Article, Event, Trajectory, ...)
    internal/web        server-rendered UI (html/template + embed.FS)
    internal/ingest     Colly crawler (Phase 2)
    internal/store      PostgreSQL layer (Phase 20)
    internal/api        REST surface (Phase 12)
    migrations/         SQL migrations
    docs/               product + technical spec

The binary embeds its templates and static assets. No Node toolchain,
no bundler, no separate frontend deploy.

## Dev

    cp configs/.env.example .env
    make db-up
    make run        # http://localhost:8080
