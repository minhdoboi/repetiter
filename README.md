# Repetiter

A multi-user language-learning app built around **one sentence at a time**. The product is a **mobile app**: you practice on the phone, including offline and on a walk. A small API behind it is a SaaS: each account has its own sentence history.

The main screen is a **Google Translate-style pair of panes**: source on the left (or top on the phone), translation on the right. Language pickers sit above each pane; either side can be changed, and a swap control reverses the pair. v1 starts with **French ↔ Vietnamese**. Type a sentence, get a translation, hear it spoken, keep an MP3 on the phone, then generate reformulations and related phrases — those suggestions sit **under the target text**. Text history lives on your account so it survives a new device; audio lives on the smartphone and can be regenerated if missing.

## Why

Reading a translation is not enough. You remember a language when you hear it, say it again, and see the same idea expressed in a few different ways. Repetiter turns a single sentence into a small practice loop: translate, listen, reformulate, save.

Listening is the core loop. That is why the client is mobile-first: cached audio on the device for replay offline. Background playback and lock-screen controls are planned; a browser tab works via Expo web but is not the primary target.

## How it works

1. Open the app (locally: a fixed dev user via `DEV_USER_ID`; production sign-in with Clerk is planned).
2. Pick source and target (default **French → Vietnamese**), or swap them.
3. Type a sentence in the left / top pane.
4. Read the translation in the right / bottom pane, with reformulations and related phrases underneath.
5. Tap the speaker icon to hear speech; audio is cached on the device (MP3/WAV in the app sandbox).
6. Tap a suggestion’s **open-in-browser** icon to look it up in Google Translate (target language → source language, e.g. vi → fr).
7. Tap a suggestion to load it as a new source sentence (languages swap automatically).
8. Browse **Recent** history, **save** sentences into named **folders**, or clear history. Reopen any entry on the Translate tab.
9. Change text and voice providers in **Settings** (OpenAI, Mistral, Groq, ElevenLabs, Google TTS, or on-device fallback).

## Interface

Two panes, language names above, swap in the middle — the same shape as Google Translate. Both pickers work; swapping also moves the text so you can go back the other way.

v1 ships **French and Vietnamese** only (both directions). More pairs later; the layout does not change.

**Phone** (primary): panes stacked. Source on top, translation under it, suggestions under the translation. Play speech from the target pane and from each suggestion chip.

**Wide layout** (tablet or Expo web on a laptop): panes side by side, suggestions under the target pane — same controls and history as on the phone.

```
[ Français ▼ ]              ⇄              [ Tiếng Việt ▼ ]
┌──────────────────────┐                  ┌──────────────────────┐
│ Type a sentence      │                  │ Translation    🔊    │
│                      │                  │                      │
└──────────────────────┘                  └──────────────────────┘
                                          Reformulations  🔊 ↗
                                          Related         🔊 ↗
```

↗ opens Google Translate for that suggestion (e.g. vi → fr).

Suggestions stay under the **target** text so the practice loop stays in the language you are learning.

## Features

### Translate

Type in the source pane; the target pane fills with a clear translation. Change either language above the panes, or hit swap. First pair: French ↔ Vietnamese.

### Speak and keep audio (on the device)

Any translated sentence or suggestion can be spoken. The API streams audio; the app writes it to local storage and replays from cache on later plays. If the file is missing or the TTS settings changed, it is fetched again. When server TTS fails, the app falls back to the OS voice (`expo-speech`).

### Reformulations and related topics

Listed **below the translation** (right pane on a wide layout, under the target text on a phone). Each translate request returns:

- **Reformulations** — the same meaning in the target language, said differently (register, word choice, or structure). Each chip has play, **Google Translate** (target → source language), and tap-to-reuse as a new source sentence.
- **Related** — nearby sentences that expand the situation (follow-ups, answers, or vocabulary around the same scene). Each chip has play, **Google Translate** (target → source language), and tap-to-reuse.

Audio for a variant is generated only when you tap its speaker icon.

### History and folders (per account)

Every sentence you translate is stored on your account as **text**, with reformulations and related variants.

- **Recent** — chronological list; tap to reopen on the Translate tab; delete individual entries; **Clear all** removes everything not saved to a folder.
- **Saved** — sentences moved into named **folders** (create folders when saving from Recent).
- Audio is a **device cache**, not synced. New phone or reinstall: text history comes back from the server; missing audio is regenerated on demand.

### Settings (per account)

Pick **text provider + model** (translation and suggestions) and **TTS provider + model + voice** independently. Available without extra keys: **mock** text and **Google TTS (free)**. API keys on the server unlock OpenAI, Mistral (EU endpoint), Groq, and ElevenLabs.

## Typical session

Default pair: French → Vietnamese. You type: *Pourriez-vous me dire comment aller à la gare ?*

The right pane shows the Vietnamese translation. You listen from a local MP3, then generate a few variants under it (*Làm sao để đến nhà ga?*, *Nhà ga đi đường nào?*) and related lines (*Có xa không?*, *Đi bộ được không?*, *Nên bắt xe buýt nào?*). Swap the languages to practice the other direction. The phrases live on your account. The audio lives on the phone, ready for the commute.

## Tech choices

Mobile-first **Expo** client + **Go API**: accounts and sentences in Postgres, TTS as generate-and-stream, audio cached on the device. Translation, suggestions, and speech over HTTP — no Python in the stack.

| Layer | Choice | Why |
|---|---|---|
| Client | Expo (React Native + web) | One codebase; stacked panes on phone, side-by-side on wide screens; local audio cache |
| API | Go (Chi) + pgx | One binary, JSON + stream audio; fits a small Fly machine in Paris |
| Auth | `DEV_USER_ID` locally; Clerk planned | Multi-user data model today; OAuth later |
| Text AI | Pluggable: mock, OpenAI, Mistral, Groq | User picks provider + model in Settings; same JSON prompt for all |
| Voice | Pluggable: Google TTS (free), OpenAI, ElevenLabs, Mistral Voxtral, Groq Orpheus | User picks TTS provider + model + voice; API streams audio |
| On-device fallback | OS voices via `expo-speech` | When server TTS fails or mock TTS is selected |
| Database | PostgreSQL | Per-user text history, folders, variants, daily usage counters; no audio blobs |
| Audio on device | Expo File System (app sandbox) | Offline replay; files keyed by sentence/variant and TTS settings |
| Rate limits | Postgres `usage_daily` | 50 translations/day and 300 TTS seconds/day per user |
| Ship (planned) | Static Go binary on Fly.io Paris (`cdg`) | Stateless API, EU Postgres |

The API is a set of authenticated routes under `/v1`:

| Method | Route | Purpose |
|---|---|---|
| `GET` | `/health` | Liveness + Postgres ping |
| `GET` | `/v1/me` | Current user and provider settings |
| `PATCH` | `/v1/me` | Update language pair and provider settings |
| `POST` | `/v1/sentences` | LLM JSON (translation + reformulations + related), persist |
| `GET` | `/v1/sentences` | List history (`scope=history`) or saved (`scope=saved`, optional `folder_id`) |
| `GET` | `/v1/sentences/{id}` | One sentence with variants |
| `PATCH` | `/v1/sentences/{id}` | Move sentence to a folder |
| `DELETE` | `/v1/sentences/{id}` | Delete one sentence |
| `DELETE` | `/v1/sentences/history` | Clear recent history (keeps folder saves) |
| `GET` | `/v1/folders` | List folders |
| `POST` | `/v1/folders` | Create folder |
| `POST` | `/v1/tts` | Stream `audio/mpeg` or `audio/wav` |
| `GET` | `/v1/tts/voices?provider=…` | List voices (OpenAI, Groq, Mistral, Google) |

**Go, not Rust.** This is JSON, JWT, Postgres, and streaming a file. Go ships faster. Rust (Axum + sqlx) is a fine later swap if you already want Rust; it is not lighter to *build*.

**Not PocketBase for v1.** PocketBase (also Go) can collapse auth + SQLite + REST into one binary for a prototype. Skip it once you want managed Postgres, Clerk, and an EU database.

A separate web companion is not required — run the same Expo app in the browser (`npx expo start --web`) for a side-by-side layout on a laptop.

No separate Translate API: the LLM returns translation, reformulations, and related lines in one JSON response.

### Audio: generate, download, forget

1. The app requests speech for a sentence (authenticated).
2. The API calls the configured TTS provider and **streams the audio** in the response.
3. The app writes the file under something like `{user_id}/{sentence_id}.mp3` in local storage.
4. The API **does not keep** the file. Postgres records that audio was generated (provider, model, language, hash of the text), not a bucket key.
5. On play: use the local file. If it is missing, request generation again.

That gives you a SaaS with accounts and sync, without becoming an audio host. Regenerating after a reinstall costs TTS; that is acceptable compared to storing every file forever.

OS voices are a fallback, not the default for learning pronunciation — packs differ by phone.

### Choose provider and model

Text and voice are separate settings. ElevenLabs is **TTS only**; it does not translate or rewrite sentences. Groq can do **text** (fast open models) and **TTS** via Orpheus (English/Arabic). OpenAI and Mistral can do both text and speech.

Each user (or plan) stores:

- `text_provider` + `text_model` — translation and suggestions
- `tts_provider` + `tts_model` (+ voice id when the provider has one)

The Go API keeps small interfaces (`TextProvider`, `TtsProvider`) and OpenAI-compatible HTTP clients where possible (OpenAI, Groq, Mistral chat). Keys stay on the server; the app only sends the chosen provider/model ids the account is allowed to use.

#### Text providers

| Provider | Role | Notes |
|---|---|---|
| **mock** | No API key | Fixed placeholder translations for UI dev |
| **OpenAI** | Quality default, strong Vietnamese | Chat Completions; models like `gpt-4o-mini` |
| **Mistral** | Europe + cheap | French company; EU endpoint `api.eu.mistral.ai` (+10% vs global). Vietnamese is listed as supported |
| **Groq** | Speed / low cost | Hosts open models (e.g. `openai/gpt-oss-20b`); US inference, not EU residency |

#### TTS providers

| Provider | Role | Notes |
|---|---|---|
| **Google TTS (free)** | No API key | Uses Google Translate’s TTS endpoint (gTTS-style); `fr` and `vi`; Normal / Slow voices |
| **OpenAI TTS** | Simple default | `gpt-4o-mini-tts`; preset voices (alloy, nova, …) |
| **ElevenLabs** | Best sounding voice | `eleven_flash_v2_5`; custom voice id |
| **Mistral Voxtral** | EU-friendly speech | `voxtral-mini-tts-2603`; voices listed from Mistral API |
| **Groq Orpheus** | Fast / cheap EN | `canopylabs/orpheus-v1-english`; not suitable for Vietnamese |
| **Mock / on-device** | Offline fallback | Server returns an error; app uses `expo-speech` |

#### Recommended defaults

| Goal | Text | Voice |
|---|---|---|
| **Europe + cheap** | Mistral **Small 4** via `api.eu.mistral.ai` (~$0.15 / $0.60 per 1M tokens, +10% EU) | Mistral Voxtral, or OpenAI TTS if Voxtral is not ready |
| **Vietnamese quality** | OpenAI **`gpt-4o-mini`** (solid Vietnamese; good enough for translate + reformulate) | ElevenLabs **`eleven_flash_v2_5`** (Vietnamese + cheaper than Multilingual), or OpenAI TTS if you want one vendor |
| **Fast / cheap sandbox** | Groq **`openai/gpt-oss-20b`** | Google TTS (free), Groq Orpheus (EN), or OpenAI TTS |
| **No API keys at all** | **mock** | **Google TTS (free)** for speech |

Ship defaults in `.env.example`: mock text + Google TTS. Add OpenAI / Mistral / Groq / ElevenLabs keys when you want neural translation or premium voices.

### Suggestions: AI-generated

One structured prompt per input sentence, returning JSON (same schema for every text provider):

```json
{
  "translation": "...",
  "reformulations": ["...", "...", "..."],
  "related": ["...", "...", "..."]
}
```

That covers translate, rephrase, and “nearby topic” in one round trip. Local models (Ollama) are an option for self-hosting; a hosted model is the default for a public SaaS.

### Data model (per user)

Every row is scoped to `user_id`. The API never returns another user’s sentences.

- `users` — identity; preferred language pair (default `fr` → `vi`); `text_provider` / `text_model` / `tts_provider` / `tts_model` / `tts_voice`; `plan`
- `sentences` — source text, translation, language pair, optional `folder_id`
- `folders` — named collections for saved sentences
- `variants` — reformulation vs related, parent sentence, text
- `audio_jobs` — provider, model, language, text hash, generated_at; **no file payload**
- `usage_daily` — translations and TTS seconds per user per day

PostgreSQL holds the rows. The phone holds the bytes. The API is stateless.

### Multi-user SaaS constraints

- **Auth on every route** (via `DEV_USER_ID` locally; Clerk JWT planned).
- **Isolation.** Queries always filter by `user_id`.
- **Cost control.** Daily caps: **50 translations** and **300 TTS seconds** per user in `usage_daily`.
- **No secrets in the client.** OpenAI and TTS credentials stay on the API. The app only stores its own files.
- **Horizontal deploy.** The API is stateless. Postgres is the source of truth for text. Local MP3s are per device.

Billing (Stripe) is not required for the first deploy. The data model should still leave room for a plan (`free` | `pro`) so allowed providers, models, and rate limits can differ later.

### Local vs production

| | Local | Production |
|---|---|---|
| Client | Expo (iOS Simulator / Android emulator / device) | App Store / Play Store |
| Database | Postgres in Docker | Managed Postgres in the EU (`ams` or `fra`) |
| Audio | App sandbox on the simulator/device | App sandbox on the user’s phone |
| Auth | `DEV_USER_ID` in `.env` | Clerk (prod instance) |
| API | `go run` against local Postgres | Static binary on Fly.io Paris (`cdg`), ~256MB |

### Run locally

Yes — local is the default way to build this. Three processes, one compose file:

```bash
cp .env.example .env            # optional: add OPENAI_API_KEY (mock works without it)
docker compose up -d postgres   # only dependency that needs Docker
cd api && go run ./cmd/api      # :8080
cd app && npx expo start        # phone: Expo Go / simulator; laptop: press `w` or `npx expo start --web`
```

Layout: `api/` is the Go service on `:8080`; `app/` is the Expo client (stacked panes on a phone, side-by-side on a wide web window).

Put API keys in `.env` (`OPENAI_API_KEY`, optionally Mistral / Groq / ElevenLabs). Without keys, the API uses **mock** text and **Google TTS** still works for speech. Switch providers in the app **Settings** tab. The Expo app talks to `http://localhost:8080` by default (`http://10.0.2.2:8080` on the Android emulator). Override with `EXPO_PUBLIC_API_URL` (your machine’s LAN IP when using a physical phone).

**Auth locally:** set `DEV_USER_ID=local-dev` (or any string) in `.env` — no sign-in screen yet.

**What you need on the machine:** Go, Node (for Expo), Docker. AI keys are optional (mock + Google TTS work without them). No Redis, no MinIO, no object storage. Audio lands in the simulator/phone/web sandbox.

## Status

### Implemented

**Client (Expo)** — three tabs: **Translate**, **History**, **Settings**.

- French ↔ Vietnamese; language pickers and swap (swap moves text and clears suggestions).
- Responsive layout: stacked panes on a phone, side-by-side on wide screens / Expo web.
- Translation + reformulations + related phrases in one request.
- TTS play on translation and each suggestion; local audio cache keyed by text and TTS settings.
- Google Translate link on each **reformulation and related** suggestion (opens target → source, e.g. vi → fr).
- Tap a suggestion to reuse it as a new source sentence.
- History: Recent list, per-item delete, clear all (keeps folder saves).
- Saved: move sentences into named folders; browse by folder.
- Settings: pick text and TTS provider/model/voice per account.

**API (Go + Postgres)**

- Chi router, CORS, health check, schema auto-applied on start.
- Pluggable text providers: **mock**, OpenAI, Mistral (EU base URL), Groq.
- Pluggable TTS providers: **Google TTS (free)**, OpenAI, ElevenLabs, Mistral Voxtral, Groq Orpheus, mock (on-device fallback).
- Voice listing for OpenAI, Groq, Mistral, Google.
- Daily rate limits (50 translations, 300 TTS seconds).
- `audio_jobs` audit log (no stored audio files).

**Local dev**

- Docker Compose Postgres, `DEV_USER_ID` auth, `docker compose up -d postgres` + `go run ./cmd/api` + `npx expo start`.

### Not yet

- Clerk sign-in (email / Google).
- Production deploy (Fly.io / EU Postgres).
- Background playback and lock-screen controls.
- Stripe / plan-gated providers.
- More language pairs beyond French and Vietnamese.
- Separate marketing site.

### Deploy from France

Target users in France: keep the **API and database** in the EU. Audio stays on the phone, so you are not hosting a voice library. Default is **Fly.io in Paris (`cdg`)** for the API and **Postgres in Amsterdam or Frankfurt** (Fly managed Postgres is not in Paris). That is “account data in Europe,” not a French sovereign cloud — Fly is still a US company.

| Goal | Where | Notes |
|---|---|---|
| Ship fast, users in France | **Fly.io** Paris (`cdg`) | Best match for the API. Pin machines to `cdg` / `ams` / `fra`. |
| French company, EU law | **Scaleway** (Paris) | Containers and managed Postgres in Paris. |
| Heroku-like PaaS, French | **Scalingo** or **Clever Cloud** | Git push, Postgres addon, Paris / Roubaix / Frankfurt. |
| Fly-like DX, EU company | **Koyeb** | French SAS, container deploys. |
| Cheap and still EU | **Hetzner** (DE) or **OVHcloud** + Docker | You operate it. OVH is French; Hetzner is cheaper. |
| Hyperscaler later | **Azure France Central** or **AWS Paris (`eu-west-3`)** | Useful when you want Azure OpenAI in-region, not for v1. |
| Apps | **Apple App Store / Google Play** (France) | Distribution for the client. |

Avoid a US-only Railway or Render instance for the first production API. Render Frankfurt is acceptable; Paris is better.

Hosting in Paris does not keep **everything** in Europe:

- **Clerk** stores identity in the US (GDPR via DPF/SCCs, no EU region today). Fine for an early SaaS. If a contract requires EU residency, use Keycloak, Zitadel, or Supabase Auth in Frankfurt.
- **OpenAI / Groq / ElevenLabs** process outside the EU by default. For EU inference on text (and optionally voice), prefer **Mistral** on `api.eu.mistral.ai`. OpenAI Europe (`eu.api.openai.com`) or Azure OpenAI EU Data Zone / France Central are alternatives when available.
- **Apple and Google** distribute the app; that is separate from where Postgres lives.

Sentences sent to the LLM/TTS are personal data when they contain names or school work. Voice **files** stay on the device; **text history** stays in EU Postgres. Keep Postgres in the EU even if auth or the LLM is elsewhere.

**If French customers or a DPA will ask where the data lives:** Scaleway Paris for the API and database; Mistral for text; audio still on the phone; Clerk until residency becomes a blocker.

## License

This project is licensed for **non-commercial use only**. You may use, modify, and share it for personal or educational purposes, but not in commercial products or services without written permission. See [LICENSE](LICENSE) for the full terms.
