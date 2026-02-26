# SearchKit Host Integration Guide

This guide defines the host-facing contract for embedding SearchKit in apps such as:

- `hentai0` (video search)
- `doujins` (gallery search)

SearchKit is a library. Hosts own HTTP routes, auth, and response shaping.

## Contract

Hosts should only use:

- `client.Search(ctx, query, searchkit.SearchOptions{...})`
- `client.Typeahead(ctx, query, searchkit.TypeaheadOptions{...})`

Use `SearchOptions.Mode`:

- `searchkit.SearchModeLexical`
- `searchkit.SearchModeSemantic`
- `searchkit.SearchModeDual`

Do not call low-level `searchkit/search` package APIs from host app code for normal request paths.

## Filter Policy (Host-Owned)

Hosts inject trusted SQL via:

- `FilterSQL string`
- `FilterArgs map[string]any`

Rules:

- Never concatenate raw user input into `FilterSQL`.
- Use named args in `FilterArgs`.
- Apply auth/business policy in host code and pass the resulting filter into SearchKit.

SearchKit applies filters in retrieval queries before ranking/pagination.

## Language Strictness

Hosts pass request language explicitly (`Language`) and choose `LanguageMode`:

- `searchkit.LanguageModeExact` (default): requested language only.
- `searchkit.LanguageModeFallbackEnglish`: requested language + English.

For strict language behavior, set `LanguageModeExact` and keep host hydration/read-model queries strict as well (no implicit English fallback).

## Example: hentai0 (video search)

```go
const videoFilterSQL = `
EXISTS (
  SELECT 1
  FROM hentai0.videos v
  JOIN hentai0.video_versions vv ON vv.video_id = v.id
  WHERE v.id::text = sd.entity_id
    AND v.deleted_at IS NULL
    AND vv.deleted_at IS NULL
    AND (vv.live_at IS NULL OR vv.live_at <= NOW())
    AND v.default_version_id::uuid = vv.id::uuid
)`

hits, err := searchkitClient.Search(ctx, query, searchkit.SearchOptions{
  Language:     language,
  LanguageMode: searchkit.LanguageModeExact,
  Mode:         searchkit.SearchModeLexical,
  EntityTypes:  []string{"video"},
  Limit:        100,
  FilterSQL:    videoFilterSQL,
})
```

Host then hydrates `EntityID` values into API response cards.

## Example: doujins (gallery typeahead)

```go
filterSQL := `
EXISTS (
  SELECT 1
  FROM doujins.galleries g
  JOIN doujins.gallery_i18n gi
    ON gi.gallery_id = g.id
   AND gi.language = @language
  LEFT JOIN doujins.gallery_i18n_versions giv
    ON giv.id = gi.default_version_id
  WHERE g.id::text = sd.entity_id
    AND (@show_soft_deleted OR g.deleted_at IS NULL)
    AND (@show_soft_deleted OR gi.deleted_at IS NULL)
    AND (@show_soft_deleted OR giv.deleted_at IS NULL OR giv.id IS NULL)
    AND (@show_drafts OR giv.live_at IS NOT NULL OR giv.id IS NULL)
    AND (@show_future OR giv.live_at <= NOW() OR giv.id IS NULL)
)`

hits, err := searchkitClient.Typeahead(ctx, query, searchkit.TypeaheadOptions{
  Language:     language,
  LanguageMode: searchkit.LanguageModeExact,
  EntityTypes:  []string{"gallery"},
  Limit:        12,
  FilterSQL:    filterSQL,
  FilterArgs: map[string]any{
    "language":          language,
    "show_soft_deleted": permissionOptions.ShowSoftDeleted,
    "show_drafts":       permissionOptions.ShowDrafts,
    "show_future":       permissionOptions.ShowFuture,
  },
})
```

Host then resolves IDs to gallery payloads and applies response formatting.

## Migration Checklist

- Create and reuse a single `searchkit.Client`.
- Remove app-local lexical backend branching (FTS/PGroonga selection).
- Move app business filtering to host filter-builders (`FilterSQL/FilterArgs`).
- Keep app code to request validation, policy mapping, and hydration.
