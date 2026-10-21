# spotify-history-cli: Relational Spotify Analytics Engine (Go)

A blazingly fast, relational analytics engine and CLI tool for exploring, querying, and analyzing your lifetime Spotify streaming history.

Built with **Go** and **pure-Go SQLite (CGO-free WAL mode)**. Ingests over 200,000 streams in **<12s**, automatically detects both Spotify Extended (`endsong_*.json`, `Streaming_History_Audio_*.json`) & Standard (`StreamingHistory_music_*.json`) export formats, deduplicates overlapping data, detects front-to-back album playthroughs, and executes analytical queries in **<5ms**.

---

## Installation & Quick Start

### Option 1: Install via `go install`
```bash
go install github.com/chrisburlacu1/spotify-history-cli@latest
```

### Option 2: Clone and Build from Source
```bash
git clone https://github.com/chrisburlacu1/spotify-history-cli.git
cd spotify-history-cli
go build -o spotify-history-cli .
```

---

## Ingesting Your Spotify History

1. **Request your data from Spotify:**
   Go to your [Spotify Account Privacy Settings](https://www.spotify.com/account/privacy/) and request either:
   - **Account data** (Standard History: `StreamingHistory_music_*.json`)
   - **Extended streaming history** (Full history: `endsong_*.json` or `Streaming_History_Audio_*.json`)
2. **Ingest your files into relational SQLite:**
   ```bash
   # Point directly to your extracted folder:
   spotify-history-cli ingest ~/Downloads/my_spotify_data/

   # Or specify individual JSON files:
   spotify-history-cli ingest endsong_0.json endsong_1.json

   # Or specify multiple folders via flag:
   spotify-history-cli ingest --dirs ./data,./extra-data
   ```

---

## Metadata Enrichment (Spotify API)

Standard Spotify exports (`StreamingHistory_music_*.json`) only provide artist, track name, and timestamp. They do **not** include album names, sequential track numbers, or release dates.

The built-in **Enricher** (`spotify-history-cli enrich`) fills these gaps by connecting to the Spotify Web API using a smart **discography-first algorithm**:
- **Batch Discography Resolution:** Instead of sending an API request for every single track (which quickly hits rate limits), the enricher queries full album catalogs for your top artists in bulk.
- **Enables Deep Analytics:** Populates missing track numbers and release years, unlocking full front-to-back album playthrough detection (`playthroughs`) and decade analysis (`eras`).
- **Resilient Rate-Limiting:** Handles Spotify API rate limits automatically by parsing `Retry-After` headers and retrying with exponential backoff.

### Setup & Usage
Create a free app at the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard) to get your credentials:

```bash
# Set credentials in your environment
export SPOTIFY_CLIENT_ID="your_client_id"
export SPOTIFY_CLIENT_SECRET="your_client_secret"

# Enrich metadata for your top 50 most-listened artists (~150 API calls)
spotify-history-cli enrich --top 50
```

---

## Relational Database Architecture

The SQLite database is normalized with foreign key constraints, indexes, and full-text search:

```
┌──────────────┐       ┌──────────────┐       ┌──────────────┐
│   artists    │◄──┬───│    albums    │◄──────│    tracks    │
└──────────────┘   │   └──────────────┘       └──────────────┘
                   │                                 ▲
                   │                                 │
                   └─────────────────────────────────┤ (track_artists)
                                                     │
                                              ┌──────────────┐
                                              │   streams    │
                                              └──────────────┘
```

- **`artists`**: Unique artist catalog and Spotify URIs.
- **`albums`**: Album titles, types (`album`, `single`), release dates, and cover URLs.
- **`tracks`**: Track titles, track numbers, and durations.
- **`streams`**: Fact table with timestamps, pre-computed date dimensions, skip flags, deduplicated by `UNIQUE(played_at, track_id, ms_played)`.
- **`v_streams` / `v_top_albums` / `v_top_artists` / `v_top_tracks`**: Instant views for SQL queries and AI agent exploration.

---

## CLI Commands & Analytics

### Global Flags
- `--year <YYYY|all>` (alias: `-y`): Filter by year (`2026` by default, `all`, `2025`, `2024`, etc.).
- `--json`: Output raw structured JSON for AI agents and scripts.
- `--no-color`: Disable ANSI terminal formatting.
- `--db <path>`: Custom SQLite database path (defaults to `./spotify.db`, `$SPOTIFY_DB`, or standard OS app config directory).

---

### 1. `overview` (aliases: `stats`, `summary`)
High-level listening metrics, peak days, skip rates, active days, and top artist/album:
```bash
spotify-history-cli overview
spotify-history-cli overview --year all
```

---

### 2. `top`
Rank your top artists, albums, and tracks by listening duration:
```bash
# Top 10 artists, albums, and tracks
spotify-history-cli top -n 10

# Only Top Albums
spotify-history-cli top --albums -n 10

# All-Time Top Albums
spotify-history-cli top --albums --year all -n 15

# Only Top Tracks
spotify-history-cli top --tracks -n 15
```

---

### 3. `album <title>`
Inspect an album's complete tracklist in sequential order, showing play counts, skips, and drop-off completion rates:
```bash
spotify-history-cli album "Blonde"
spotify-history-cli album "Stick Season"
```

---

### 4. `playthroughs` (aliases: `album-loops`, `marathons`)
Detects continuous sequential front-to-back album listens (Track $1 \to 2 \to 3 \dots \to N$ played in order without skipping):
```bash
spotify-history-cli playthroughs --min 4
spotify-history-cli playthroughs --year all
```

---

### 5. `eras` (aliases: `decades`, `vintage`)
Analyzes your listening breakdown by music release decade (2020s, 2010s, 2000s, 90s, etc.) with horizontal ASCII bar charts:
```bash
spotify-history-cli eras
spotify-history-cli eras --year all
```

---

### 6. `deepdive <artist>`
Full multi-year profile for an artist: timeline, catalog breadth, monthly listening bar chart, top albums, top tracks, and peak listening hour:
```bash
spotify-history-cli deepdive "Gracie Abrams"
spotify-history-cli deepdive "Frank Ocean"
spotify-history-cli deepdive "Noah Kahan"
```

---

### 7. `habits` (aliases: `patterns`, `time`)
Behavioral habits:
- **24-hour distribution** (when you listen)
- **Day-of-week distribution**
- **Continuous listening session detection** (unbroken listening marathons with top album/artist per session)
```bash
spotify-history-cli habits
spotify-history-cli habits --year all
```

---

### 8. `obsessions` (aliases: `loops`, `binges`)
Identifies single-day repeat binges (songs played $\ge 5$ times in a single 24-hour window):
```bash
spotify-history-cli obsessions -m 5 -n 10
```

---

### 9. `wrapped`
Generates an interactive terminal Spotify Wrapped recap with listening archetype, peak month/day, top artists, albums, tracks, and top obsession:
```bash
spotify-history-cli wrapped
spotify-history-cli wrapped --year all
```

---

### 10. `enrich` (Spotify Web API)
Fetches missing album metadata, release dates, track sequencing, and cover art:
```bash
spotify-history-cli enrich --top 50
```

---

### 11. `search <query>`
Full-Text Search (FTS5) across all tracks, artists, and albums:
```bash
spotify-history-cli search "Frank Ocean"
spotify-history-cli search "Ribbons"
```

---

### 12. `sql "<query>"`
Execute arbitrary SQL directly with table or JSON output:
```bash
# Custom multi-table query
spotify-history-cli sql "SELECT artist_name, album_name, COUNT(*) as plays FROM v_streams WHERE year = 2026 GROUP BY 1, 2 ORDER BY plays DESC LIMIT 5"

# Inspect database schema
spotify-history-cli sql --schema
```

---

### 13. `version`
Print build and version metadata:
```bash
spotify-history-cli version
spotify-history-cli version --json
```

---

### 14. `completion <shell>`
Generate shell autocompletion script (bash, zsh, fish, powershell):
```bash
# Zsh
spotify-history-cli completion zsh > "${fpath[1]}/_spotify-history-cli"

# Bash
spotify-history-cli completion bash > /etc/bash_completion.d/spotify-history-cli
```

---

## AI Agent Integration (`--json`)

Every command supports `--json` for machine consumption:

```bash
spotify-history-cli overview --json
spotify-history-cli top --albums --json
spotify-history-cli wrapped --year all --json
spotify-history-cli sql "SELECT * FROM v_top_albums LIMIT 5" --json
```

---

## License

MIT © [Chris Burlacu](https://github.com/chrisburlacu1)
