# Repetiter

A multi-user language-learning app built around **one sentence at a time**. The product is a **mobile app**: you practice on the phone, including offline and on a walk. A small API behind it is a SaaS: each account has its own sentence history.

Type a sentence the way you would in Google Translate. Get a translation, hear it spoken, keep an MP3 on the phone, then generate reformulations and related phrases. Text history lives on your account so it survives a new device; audio lives on the smartphone and can be regenerated if missing.

## Why

Reading a translation is not enough. You remember a language when you hear it, say it again, and see the same idea expressed in a few different ways. Repetiter turns a single sentence into a small practice loop: translate, listen, reformulate, save.

Listening is the core loop. That is why the client is mobile: background playback, lock-screen controls, and files in the phone’s storage. A browser tab is a weak Walkman.

## How it works

1. Sign in on the phone (email or Google).
2. Type a sentence in the source language.
3. Get a translation in the target language.
4. Generate speech; the MP3 is downloaded into the app’s storage and played from the device.
5. Ask for reformulations of the same idea, or related sentences on the same topic.
6. Find every sentence again later in your history. Replay from local files, or regenerate audio if the file is gone.

## Features

### Translate

A simple, focused input. You type a sentence; the app returns a clear translation. No extra chrome — just the text you need to practice.

### Speak and keep as MP3 (on the phone)

Any translated sentence can be converted to speech. The file is stored in the app sandbox so you can replay it offline, on a walk, or export it to another player. The API generates the bytes; it does not keep a library of everyone’s audio.

### Reformulations and related topics

From one sentence, generate:

- **Reformulations** — the same meaning, said differently (register, word choice, or structure).
- **Related topics** — nearby sentences that expand the situation (follow-ups, answers, or vocabulary around the same scene).

Use these to move from “I understood this line” to “I can say this idea in more than one way.” Audio for a variant is generated only when you play or save it.

### History (per account)

Every sentence you translate is stored on your account as **text**. Browse the list, jump back in, generate more reformulations. Users never see each other’s sentences.

MP3s are a **device cache**, not the source of truth. New phone or reinstall: history comes back from the server; missing files are generated again on demand.

## Typical session

You type: *Could you tell me how to get to the station?*

You get a translation, listen to it from a local MP3, then generate a few variants (*What’s the way to the station?*, *How do I reach the train station from here?*) and related lines (*Is it far?*, *Can I walk?*, *Which bus should I take?*). The phrases live on your account. The audio lives on the phone, ready for the commute.

## Tech choices

Mobile-first client, slim API: accounts and sentences in Postgres, TTS as generate-and-download, MP3s only on the device.

| Layer | Choice | Why |
|---|---|---|
| Client | Expo (React Native) | One codebase for iOS and Android, file system, background audio, App Store / Play Store |
| API | FastAPI | Native fit for TTS, streaming an MP3 once, and LLM SDKs |
| Auth | Clerk (email + Google) | Multi-user from day one without building password reset, OAuth, or sessions |
| Text AI | OpenAI (`gpt-4o-mini`) | Translation, reformulations, and related sentences in one JSON call |
| Voice | gTTS default, OpenAI TTS optional | Server generates an MP3; the phone stores it. Neural voice when pronunciation quality matters |
| On-device fallback | OS voices (iOS / Android TTS) | Offline speak when the network is down; quality varies by language pack |
| Database | PostgreSQL | Per-user text history; no audio blobs |
| Audio on device | Expo File System (app sandbox) | Offline replay, no object-storage bill, voice files stay with the user |
| Temp audio (optional) | Short-lived file on the API or R2 TTL | Only if you cannot stream the MP3 in the same response |
| Cache / limits | Redis | Rate-limit TTS and LLM calls so one user cannot burn the API bill |
| Ship | Docker + Fly.io in Paris (`cdg`) | Stateless API, EU Postgres. Stores are not in the cloud |

A small **web companion** (React) can stay for typing long sentences on a laptop. It syncs the same history; audio is generated when the user opens the phone.

No separate Translate API at first. The LLM already writes reformulations, so it can translate in the same request and stay consistent.

### Audio: generate, download, forget

1. The app requests speech for a sentence (authenticated).
2. The API runs TTS and **streams the MP3** in the response (or puts it on a TTL URL for a few minutes).
3. The app writes the file under something like `{user_id}/{sentence_id}.mp3` in local storage.
4. The API **does not keep** the file. Postgres records that audio was generated (provider, language, hash of the text), not a bucket key.
5. On play: use the local file. If it is missing, request generation again.

That gives you a SaaS with accounts and sync, without becoming an audio host. Regenerating after a reinstall costs TTS; that is acceptable compared to storing every file forever.

A small `TtsProvider` interface (`gtts` | `openai`) keeps the rest of the app unchanged. OS voices are a fallback, not the default for learning pronunciation — packs differ by phone.

SaaS angle later: free accounts use gTTS; paid accounts get neural voice and higher generation limits.

### Voice: gTTS vs AI

**gTTS** is the default: free, a few lines of code, writes MP3 directly. Fine for a free tier and for languages Google Translate covers. It sounds like classic Translate — usable, not a model for pronunciation. It talks to an unofficial Google endpoint, so it can break.

**OpenAI TTS** (`gpt-4o-mini-tts` or `tts-1`) is the upgrade: natural rhythm and stress, stable API, voice and speed control. That is what you want to imitate when learning. It costs per character.

### Suggestions: AI-generated

One structured prompt per input sentence, returning JSON:

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

- `users` — identity from Clerk (or a local user id in development)
- `sentences` — source text, translation, language pair, timestamps
- `variants` — reformulation vs related, parent sentence, text
- `audio_jobs` (optional) — last provider, text hash, generated_at; **no file payload**

PostgreSQL holds the rows. The phone holds the bytes. The API is stateless.

### Multi-user SaaS constraints

- **Auth on every route.** History, generate, and TTS are authenticated. Marketing site is public; the app is not.
- **Isolation.** Queries always filter by `user_id`.
- **Cost control.** Per-user daily caps on translations, reformulations, and TTS seconds. Redis enforces them. Regenerating a missing MP3 counts against the cap.
- **No secrets in the client.** OpenAI and TTS credentials stay on the API. The app only stores its own files.
- **Horizontal deploy.** The API is stateless. Postgres is the source of truth for text. Local MP3s are per device.

Billing (Stripe) is not required for the first deploy. The data model should still leave room for a plan (`free` | `pro`) so gTTS vs neural TTS and rate limits can differ later.

### Local vs production

| | Local | Production |
|---|---|---|
| Client | Expo (iOS Simulator / Android emulator) | App Store / Play Store |
| Database | Postgres in Docker | Managed Postgres in the EU (`ams` or `fra`) |
| Audio | App sandbox on the simulator/device | App sandbox on the user’s phone |
| Auth | Clerk (dev instance) | Clerk (prod instance) |
| API | `docker compose up` | Fly.io Paris (`cdg`) |

### Deploy from France

Target users in France: keep the **API and database** in the EU. MP3s stay on the phone, so you are not hosting a voice library. Default is **Fly.io in Paris (`cdg`)** for the API and **Postgres in Amsterdam or Frankfurt** (Fly managed Postgres is not in Paris). That is “account data in Europe,” not a French sovereign cloud — Fly is still a US company.

| Goal | Where | Notes |
|---|---|---|
| Ship fast, users in France | **Fly.io** Paris (`cdg`) | Best match for the API. Pin machines to `cdg` / `ams` / `fra`. |
| French company, EU law | **Scaleway** (Paris) | Containers and managed Postgres in Paris. |
| Heroku-like PaaS, French | **Scalingo** or **Clever Cloud** | Git push, Postgres + Redis addons, Paris / Roubaix / Frankfurt. |
| Fly-like DX, EU company | **Koyeb** | French SAS, container deploys. |
| Cheap and still EU | **Hetzner** (DE) or **OVHcloud** + Docker | You operate it. OVH is French; Hetzner is cheaper. |
| Hyperscaler later | **Azure France Central** or **AWS Paris (`eu-west-3`)** | Useful when you want Azure OpenAI in-region, not for v1. |
| Apps | **Apple App Store / Google Play** (France) | Distribution for the client. |

Avoid a US-only Railway or Render instance for the first production API. Render Frankfurt is acceptable; Paris is better.

Hosting in Paris does not keep **everything** in Europe:

- **Clerk** stores identity in the US (GDPR via DPF/SCCs, no EU region today). Fine for an early SaaS. If a contract requires EU residency, use Keycloak, Zitadel, or Supabase Auth in Frankfurt.
- **OpenAI’s default API** processes in the US (prompts include the sentences). Longer-term options from France: **Mistral** (French, EU-hosted), **Azure OpenAI** with EU Data Zone / France Central, or OpenAI’s Europe endpoint (`eu.api.openai.com`) if available.
- **gTTS** talks to Google unofficially. Usable for a prototype; a poor processor story for a real GDPR file. Neural TTS via Mistral, Azure, or OpenAI EU is the cleaner production path.
- **Apple and Google** distribute the app; that is separate from where Postgres lives.

Sentences sent to the LLM/TTS are personal data when they contain names or school work. Voice **files** stay on the device; **text history** stays in EU Postgres. Keep Postgres in the EU even if auth or the LLM is elsewhere.

**Now:** Expo app + Fly.io Paris + EU Postgres + Clerk + OpenAI (or Mistral if you want a French vendor from day one). No R2 library.

**If French customers or a DPA will ask where the data lives:** Scaleway Paris for the API and database; Mistral for text; audio still on the phone; Clerk until residency becomes a blocker.

## Status

Early stage. The product is mobile-first with a slim SaaS API; implementation is next.
