---
date: 2026-05-14
topic: gpt-image-2-size-tier-development-options
focus: TryValo gpt-image-2 size, ratio, edit, and billing support in new-api
---

# Ideation: gpt-image-2 Size Tier Development Options

## Codebase Context

- The OpenAI image relay surface is `POST /v1/images/generations` and `POST /v1/images/edits`.
- `dto.ImageRequest` carries `Size string`; there is no first-class `aspect_ratio` field for the OpenAI image path.
- For OpenAI-compatible channels, generation requests are passed through as JSON, and edit requests forward multipart non-file fields plus `image` / `mask` file fields.
- `GetAndValidOpenAIImageRequest` restricts sizes for `dall-e` and `dall-e-3`, but not for `gpt-image-*`.
- `ImageHelper` currently applies `n` through `OtherRatio("n")`; it does not add a 1K/2K/4K size multiplier for `gpt-image-2`.
- `tiered_expr` can read JSON request bodies through `param("size")`, but multipart edit bodies are not available to expressions unless `BillingRequestInput` is explicitly built from the parsed `ImageRequest`.

## Verified Behavior

Live tests against `https://api.tryvalo.com` showed:

- `GET /v1/models` exposes `gpt-image-2`.
- Generation and edit both accept concrete `size` strings.
- The full verified matrix is:
  - 1K: `1024x1024`
  - 2K: `1536x1024`, `1024x1536`, `1792x1024`, `1024x1792`, `2048x2048`, `2048x1152`, `1152x2048`
  - 4K: `3840x2160`, `2160x3840`
- 4K 4:3 is not supported on this upstream:
  - `3840x2880` returns `Invalid size ... Requested resolution exceeds the current pixel budget`
  - `2880x3840` returns the same class of upstream 400 error
- A `1024x1024` input image can be edited into 2K and 4K outputs. For edits, `size` means output size, not input image size.
- Large edit tests need a long client timeout; PowerShell/.NET `HttpClient`'s default 100 second timeout can produce false failures.

## Topic Axes

- Customer request contract: which parameters and sizes customers should send.
- Billing and enforcement: how 1K/2K/4K should be charged and whether mismatches are blocked.
- Provider compatibility: how much logic should be specific to TryValo/sub2api vs reusable for future image upstreams.
- Operations and testing: how to verify upstream behavior without leaking keys or misreading timeouts.

## Ranked Ideas

### 1. Configuration-only launch with model aliases (fastest)

**Description:**
Expose three sellable model names and map them to the same upstream model:

- `gpt-image-2-1k` -> upstream `gpt-image-2`
- `gpt-image-2-2k` -> upstream `gpt-image-2`
- `gpt-image-2-4k` -> upstream `gpt-image-2`

Set different fixed model prices for those aliases in `new-api`. Customer docs say which `size` values are allowed for each alias.

**Why it matters:**
This can likely ship without touching code. It is enough if the current priority is to sell quickly and customers are trusted or controlled.

**Downside:**
It does not enforce size/model consistency. A customer could call `gpt-image-2-1k` with `3840x2160`, and the gateway would forward it unless upstream rejects it.

**Status:** Recommended for immediate pilot only.

### 2. Add gateway-side size tier validation for image aliases

**Description:**
Implement a small shared size classifier for image requests:

```text
1024x1024 -> 1k
1536x1024, 1024x1536, 1792x1024, 1024x1792,
2048x2048, 2048x1152, 1152x2048 -> 2k
3840x2160, 2160x3840 -> 4k
```

Then enforce alias compatibility:

- `gpt-image-2-1k` only accepts 1K sizes
- `gpt-image-2-2k` only accepts 2K sizes
- `gpt-image-2-4k` only accepts 4K sizes

Reject unsupported ratios such as `3840x2880` / `2880x3840` before sending upstream.

**Why it matters:**
This closes the biggest commercial hole in the alias approach: customers cannot underpay by choosing a cheap alias with an expensive output size.

**Downside:**
Requires code changes in request validation or relay setup, plus tests for both JSON generation and multipart edits.

**Status:** Recommended first code change if strict billing matters.

### 3. Add size-tier OtherRatio for image billing

**Description:**
Instead of relying on aliases, derive `image_size_tier` from `request.Size` inside `ImageHelper` or a helper called from it, then add a billing multiplier to `PriceData.OtherRatios`:

```text
image_size_tier=1k -> 1.0
image_size_tier=2k -> configured multiplier
image_size_tier=4k -> configured multiplier
```

Keep `n` as the existing separate multiplier.

**Why it matters:**
This lets one public model name, `gpt-image-2`, charge different prices by output size.

**Downside:**
The multipliers need a configuration surface. Hardcoding them would be fast but less maintainable. It also needs careful log display so admins can see why a request cost more.

**Status:** Good long-term approach if the product should expose one model and bill by request fields.

### 4. Make tiered_expr request-aware for multipart image edits

**Description:**
For image requests, build `BillingRequestInput` from the parsed `dto.ImageRequest` rather than only reading raw JSON bodies. That makes expressions such as `param("size")` work for both generation JSON and edit multipart requests.

Then an admin could configure pricing like:

```text
param("size") == "1024x1024"
  ? tier("1k", 1.0)
  : (param("size") == "3840x2160" || param("size") == "2160x3840")
    ? tier("4k", 4.0)
    : tier("2k", 2.0)
```

**Why it matters:**
This reuses the existing dynamic billing framework instead of inventing a separate image-pricing system.

**Downside:**
The current expression system is token-price oriented. Per-call image pricing through expressions can work, but it must be tested carefully because image `usage` is often synthetic (`TotalTokens = 1` in this flow).

**Status:** Strong if the site already wants dynamic pricing rules in the admin UI.

### 5. Add an official image compatibility test harness

**Description:**
Promote the ad hoc PowerShell tests into durable scripts:

- generation matrix test
- edit matrix test
- dimension verification
- unsupported-size negative tests
- long timeout defaults

Keep the API key in environment variables only.

**Why it matters:**
The most expensive bugs here are false assumptions about upstream behavior. A repeatable harness lets operators validate a new upstream before changing code or pricing.

**Downside:**
Does not enforce production behavior by itself.

**Status:** Recommended alongside any code or config rollout.

## Recommended Development Plan

### Phase 1: Ship a controlled pilot without code changes

Use model aliases and fixed prices:

| Public model | Upstream model | Allowed sizes |
|---|---|---|
| `gpt-image-2-1k` | `gpt-image-2` | `1024x1024` |
| `gpt-image-2-2k` | `gpt-image-2` | `1536x1024`, `1024x1536`, `1792x1024`, `1024x1792`, `2048x2048`, `2048x1152`, `1152x2048` |
| `gpt-image-2-4k` | `gpt-image-2` | `3840x2160`, `2160x3840` |

Do not advertise 4K 4:3. It failed upstream with pixel-budget errors.

### Phase 2: Add strict validation before wider resale

Implement:

- `classifyImageSize(size string) (tier, ratio string, ok bool)`
- `expectedTierFromImageModelAlias(model string) (tier string, ok bool)`
- validation for both `/v1/images/generations` JSON and `/v1/images/edits` multipart
- clear 400 errors when alias tier and `size` mismatch
- clear 400 errors for unsupported sizes such as `3840x2880`

Suggested test cases:

- generation accepts every verified size
- edit accepts every verified size
- `gpt-image-2-1k` + `3840x2160` is rejected
- `gpt-image-2-4k` + `1024x1024` is rejected
- `3840x2880` and `2880x3840` are rejected
- multipart edit preserves `size` and validates it the same as JSON

### Phase 3: Add first-class size-tier billing

Choose one:

- If you want minimum code and explicit SKUs: keep aliases and fixed prices.
- If you want one model name with automatic pricing: add image size tier multipliers or make `tiered_expr` reliably see parsed image request fields for both generation and edit.

For this repo, the cleanest maintainable path is:

1. structured size classifier
2. strict validation
3. expose parsed image request fields to billing
4. let admin choose alias pricing or expression pricing

## Rejection Summary

| Idea | Reason Rejected |
|---|---|
| Advertise arbitrary ratios like `4K 4:3` | TryValo rejected `3840x2880` and `2880x3840` with pixel-budget errors |
| Let customers send `size: "16:9"` | The OpenAI-compatible image path expects concrete `widthxheight` strings and currently forwards `size` as-is |
| Depend only on upstream rejection | Customers can still select a cheaper alias with an expensive valid size; upstream will accept it and billing will be wrong |
| Modify sub2api | The upstream is not under local control and the user explicitly prefers not to change it |
| Hardcode 1K/2K/4K prices directly in request handling | Fast but brittle; use aliases/config first, then structured classifier plus configurable billing |

## Session Log

- 2026-05-14: Verified TryValo `gpt-image-2` generation and edit size matrix.
- 2026-05-14: Verified 4K 4:3 and 3:4 fail with upstream pixel-budget errors.
- 2026-05-14: Initial development-option ideation documented.
