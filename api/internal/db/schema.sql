CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  source_lang TEXT NOT NULL DEFAULT 'fr',
  target_lang TEXT NOT NULL DEFAULT 'vi',
  text_provider TEXT NOT NULL DEFAULT 'openai',
  text_model TEXT NOT NULL DEFAULT 'gpt-4o-mini',
  tts_provider TEXT NOT NULL DEFAULT 'openai',
  tts_model TEXT NOT NULL DEFAULT 'gpt-4o-mini-tts',
  tts_voice TEXT,
  plan TEXT NOT NULL DEFAULT 'free',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sentences (
  id UUID PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  source_text TEXT NOT NULL,
  translation TEXT NOT NULL,
  source_lang TEXT NOT NULL,
  target_lang TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sentences_user_created_idx
  ON sentences (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS folders (
  id UUID PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, name)
);

CREATE INDEX IF NOT EXISTS folders_user_idx ON folders (user_id, name);

ALTER TABLE sentences ADD COLUMN IF NOT EXISTS folder_id UUID REFERENCES folders(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS sentences_folder_idx ON sentences (user_id, folder_id, created_at DESC);

CREATE TABLE IF NOT EXISTS variants (
  id UUID PRIMARY KEY,
  sentence_id UUID NOT NULL REFERENCES sentences(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK (kind IN ('reformulation', 'related')),
  text TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS variants_sentence_idx ON variants (sentence_id);

CREATE TABLE IF NOT EXISTS audio_jobs (
  id UUID PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  sentence_id UUID REFERENCES sentences(id) ON DELETE SET NULL,
  variant_id UUID REFERENCES variants(id) ON DELETE SET NULL,
  provider TEXT NOT NULL,
  model TEXT NOT NULL,
  language TEXT NOT NULL,
  text_hash TEXT NOT NULL,
  generated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS usage_daily (
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  day DATE NOT NULL,
  translations INT NOT NULL DEFAULT 0,
  tts_seconds INT NOT NULL DEFAULT 0,
  PRIMARY KEY (user_id, day)
);
