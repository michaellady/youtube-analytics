---
name: clip-results
description: Use when the user wants the closed-loop performance of an OpusClip clip batch across all the platforms it was posted to — "clip results", "how did the clips do", "cross-platform clip performance", "closed loop on the clips", "did the clips work". Pulls per-platform metrics natively (no API exists) and renders one consolidated report.
user_invocable: true
---

# clip-results

Closed-loop measurement for a posted OpusClip batch. Given one or more OpusClip
**project** IDs/URLs, gather each clip's performance on every platform it was
posted to (YouTube, Facebook, Instagram, TikTok, LinkedIn) and produce one
consolidated cross-platform report.

## The reliability problem this solves

**There is NO programmatic analytics path.** The OpusClip API has no
analytics/insights endpoints; the postIds it stores are OpusClip-internal (can't
deep-link to the native post); and these posts bypass Buffer, so Buffer stats
don't see them. The ONLY way to get the numbers is each platform's **native
analytics, read in-browser**, matched to clips by the **`[opus:CLIPID]` tag**
embedded in every caption.

Per the Primitive Test: browser navigation/extraction stays here in the prompt
(UIs change — a script of selectors would rot; the Bitter Lesson says the model
drives changing UIs/logins better). The deterministic work — the YouTube
`[opus:]`→metrics join, aggregation, ranking, and report render — is the Go
binary `clip-results report`.

## When to Use

Use when the user asks how a *posted* clip batch performed across platforms.

Do NOT use when:
- The user wants to *create/curate/schedule* clips → that's the `opusclip` skill.
- Only YouTube numbers are wanted → `/yt-analytics` covers that (but note: YouTube
  is typically <⅓ of true cross-platform reach — see findings).

## Inputs

`$1...` — OpusClip project IDs or `clip.opus.pro/clip/<id>` URLs. The 12-char id
in that URL (e.g. `P3051823ab0w`) is the **project** id, not a single clip. One
project = one source video's clips.

## Steps

### 1. Refresh YouTube data (the join uses current metrics)
```bash
cd /Users/mikelady/dev/youtube_analytics
go run . fetch && go run . fetch-analytics --all   # skip if videos.json is fresh today
```

### 2. Clip inventory per project
The OpusClip CLI is bundled in the plugin cache:
```bash
CLI="$(ls -d /Users/mikelady/.claude/plugins/cache/opus-skills/opusclip/*/scripts/opusclip | tail -1)"
"$CLI" list --project <PROJECT_ID> --summary    # clip_id, title, score, duration
```
Capture clip_ids + titles. (Confirm OPUSCLIP_API_KEY is set.)

### 3. Scaffold the results file
Create `data/opus_clips/<PROJECT_ID>-platform-results.json`:
```json
{ "project": "<PROJECT_ID>", "source_video": "<title (videoId)>",
  "measured_at": "<YYYY-MM-DD>", "rows": [] }
```
You'll append one row per (clip × platform) as you read each dashboard. **Do not
add youtube rows by hand** — the Go tool joins them in step 5.

### 4. Browser pass — read each platform's native analytics
Load chrome tools first: `ToolSearch select:mcp__claude-in-chrome__tabs_context_mcp,mcp__claude-in-chrome__navigate,mcp__claude-in-chrome__get_page_text,mcp__claude-in-chrome__browser_batch,mcp__claude-in-chrome__computer`.

**Use `get_page_text`, not screenshots** — it returns the full caption (with the
`[opus:CLIPID]` tag for exact matching) plus the metric columns, far faster and
token-cheaper. Append rows `{opus_id, title?, platform, views, likes, comments, follows}`.
Platform string ∈ `facebook|instagram|tiktok|linkedin_org|linkedin_personal`
(youtube is auto-joined).

- **Meta Business Suite (covers BOTH Facebook + Instagram)** —
  `business.facebook.com/latest/posts/published_posts/` → Content. Resize window
  wide (~1920). **Columns** → add **Views** + **Follows**, remove Reach
  (deprecated) + Shares. The DOM appends rows as you scroll; `get_page_text`
  returns all loaded rows. IG posts show the full caption + `[opus:]` tag; FB
  posts show a short title only — match an FB post to its clip by the **same
  publish time-slot** as the tagged IG post (they're scheduled together, ~2–4 min
  apart). FB ≫ IG on views, consistently.
- **TikTok** — `tiktok.com/tiktokstudio/content` → Views/Likes/Comments columns;
  captions carry the `[opus:]` tag. **Virtualized list** — scroll in modest
  increments and dedupe by tag (big jumps skip rows). Highest engagement of any
  platform.
- **LinkedIn — EVC org** — `linkedin.com/company/<ORG_ID>/admin/analytics/updates/`
  → Content engagement table (impressions per post). Usually negligible.
- **LinkedIn — Mike personal** — Me menu → Posts & Activity →
  `linkedin.com/in/<vanity>/recent-activity/all/`; each post shows "N impressions".
  (An OpusClip `hasConflict: true` flag does NOT mean the post failed — verify.)

If a platform nav is permission-denied, ask the user before retrying; don't loop.

### 5. Consolidate (deterministic — the Go binary)
```bash
go run . clip-results report \
  --results data/opus_clips/<PROJECT_ID>-platform-results.json \
  --out     data/opus_clips/<PROJECT_ID>-platform-results.md
```
It auto-joins YouTube rows from `videos.json` by `[opus:]` tag, then computes
per-platform totals + ranking, top clip per platform (the breakout view),
cross-platform per-clip reach, and grand total — and writes the markdown.
**Never hand-compute the totals.**

### 6. Report to the user
Lead with: cross-platform grand total **vs the YouTube-only number** (YouTube is
usually a minority of true reach); platform ranking; the **platform-specific
breakouts** (top clip differs per platform → cross-posting earns its keep);
conversion (follows/subs — typically ≈0, clips buy reach not subscribers); and
that OpusClip's per-clip score is a weak predictor of real views.

### 7. Commit
`data/opus_clips/*` is gitignored, so the results files stay local. Commit only
code/skill changes.

## Common Mistakes

- Treating the OpusClip API or Buffer as an analytics source — neither reports
  post performance. Native dashboards only.
- Using screenshots instead of `get_page_text` — slower, and you lose the clean
  `[opus:]` tags that make matching exact.
- Hand-summing platform totals — use `clip-results report` (it's tested).
- Forgetting `fetch-analytics --all` first — the YouTube join then uses stale views.
- Assuming a single platform's hero clip is THE winner — breakouts are
  platform-specific.

## Known findings (baseline: P3051823ab0w, measured 2026-05-30)
Facebook was the #1 channel (beat YouTube); true reach ≈3.4× the YouTube-only
figure; breakouts platform-specific; conversion ≈0; Opus score a weak predictor.
Full data: `data/opus_clips/P3051823ab0w-platform-results.md`. See also the
project memory `project-opusclip-platform-reach`.

## This skill lives in the repo
Symlinked `~/.claude/skills/clip-results` → `/Users/mikelady/dev/youtube_analytics/skill/clip-results`.
Edit in the repo. The Go transport is `cmd_clip_results.go` (+ `_test.go`).
