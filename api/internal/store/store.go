package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type User struct {
	ID           string    `json:"id"`
	SourceLang   string    `json:"source_lang"`
	TargetLang   string    `json:"target_lang"`
	TextProvider string    `json:"text_provider"`
	TextModel    string    `json:"text_model"`
	TTSProvider  string    `json:"tts_provider"`
	TTSModel     string    `json:"tts_model"`
	TTSVoice     string    `json:"tts_voice,omitempty"`
	Plan         string    `json:"plan"`
	CreatedAt    time.Time `json:"created_at"`
}

type Sentence struct {
	ID             uuid.UUID  `json:"id"`
	UserID         string     `json:"-"`
	SourceText     string     `json:"source_text"`
	Translation    string     `json:"translation"`
	SourceLang     string     `json:"source_lang"`
	TargetLang     string     `json:"target_lang"`
	FolderID       *uuid.UUID `json:"folder_id,omitempty"`
	FolderName     string     `json:"folder_name,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	Reformulations []Variant  `json:"reformulations,omitempty"`
	Related        []Variant  `json:"related,omitempty"`
}

type Folder struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Variant struct {
	ID   uuid.UUID `json:"id"`
	Kind string    `json:"kind"`
	Text string    `json:"text"`
}

type Usage struct {
	Translations int `json:"translations"`
	TTSSeconds   int `json:"tts_seconds"`
}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) UpsertUser(ctx context.Context, id, textProv, textModel, ttsProv, ttsModel, ttsVoice string) (User, error) {
	const q = `
INSERT INTO users (id, text_provider, text_model, tts_provider, tts_model, tts_voice)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
ON CONFLICT (id) DO UPDATE SET updated_at = now()
RETURNING id, source_lang, target_lang, text_provider, text_model, tts_provider, tts_model,
          COALESCE(tts_voice, ''), plan, created_at`
	var u User
	err := s.pool.QueryRow(ctx, q, id, textProv, textModel, ttsProv, ttsModel, ttsVoice).Scan(
		&u.ID, &u.SourceLang, &u.TargetLang, &u.TextProvider, &u.TextModel,
		&u.TTSProvider, &u.TTSModel, &u.TTSVoice, &u.Plan, &u.CreatedAt,
	)
	return u, err
}

func (s *Store) GetUser(ctx context.Context, id string) (User, error) {
	const q = `
SELECT id, source_lang, target_lang, text_provider, text_model, tts_provider, tts_model,
       COALESCE(tts_voice, ''), plan, created_at
FROM users WHERE id = $1`
	var u User
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.SourceLang, &u.TargetLang, &u.TextProvider, &u.TextModel,
		&u.TTSProvider, &u.TTSModel, &u.TTSVoice, &u.Plan, &u.CreatedAt,
	)
	return u, err
}

func (s *Store) UpdateUser(ctx context.Context, id string, patch User) (User, error) {
	const q = `
UPDATE users SET
  source_lang = COALESCE(NULLIF($2, ''), source_lang),
  target_lang = COALESCE(NULLIF($3, ''), target_lang),
  text_provider = COALESCE(NULLIF($4, ''), text_provider),
  text_model = COALESCE(NULLIF($5, ''), text_model),
  tts_provider = COALESCE(NULLIF($6, ''), tts_provider),
  tts_model = COALESCE(NULLIF($7, ''), tts_model),
  tts_voice = COALESCE(NULLIF($8, ''), tts_voice),
  updated_at = now()
WHERE id = $1
RETURNING id, source_lang, target_lang, text_provider, text_model, tts_provider, tts_model,
          COALESCE(tts_voice, ''), plan, created_at`
	var u User
	err := s.pool.QueryRow(ctx, q, id, patch.SourceLang, patch.TargetLang, patch.TextProvider, patch.TextModel, patch.TTSProvider, patch.TTSModel, patch.TTSVoice).Scan(
		&u.ID, &u.SourceLang, &u.TargetLang, &u.TextProvider, &u.TextModel,
		&u.TTSProvider, &u.TTSModel, &u.TTSVoice, &u.Plan, &u.CreatedAt,
	)
	return u, err
}

func (s *Store) InsertSentence(ctx context.Context, sent Sentence, reformulations, related []string) (Sentence, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Sentence{}, err
	}
	defer tx.Rollback(ctx)

	sent.ID = uuid.New()
	const q = `
INSERT INTO sentences (id, user_id, source_text, translation, source_lang, target_lang)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING created_at`
	if err := tx.QueryRow(ctx, q, sent.ID, sent.UserID, sent.SourceText, sent.Translation, sent.SourceLang, sent.TargetLang).Scan(&sent.CreatedAt); err != nil {
		return Sentence{}, err
	}

	reform, err := insertVariants(ctx, tx, sent.ID, sent.UserID, "reformulation", reformulations)
	if err != nil {
		return Sentence{}, err
	}
	rel, err := insertVariants(ctx, tx, sent.ID, sent.UserID, "related", related)
	if err != nil {
		return Sentence{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Sentence{}, err
	}
	sent.Reformulations = reform
	sent.Related = rel
	return sent, nil
}

func insertVariants(ctx context.Context, tx pgx.Tx, sentenceID uuid.UUID, userID, kind string, texts []string) ([]Variant, error) {
	out := make([]Variant, 0, len(texts))
	const q = `INSERT INTO variants (id, sentence_id, user_id, kind, text) VALUES ($1, $2, $3, $4, $5)`
	for _, t := range texts {
		if t == "" {
			continue
		}
		v := Variant{ID: uuid.New(), Kind: kind, Text: t}
		if _, err := tx.Exec(ctx, q, v.ID, sentenceID, userID, kind, t); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *Store) ListSentences(ctx context.Context, userID string, limit int, scope string, folderID *uuid.UUID) ([]Sentence, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	switch scope {
	case "", "history":
		scope = "history"
	case "saved", "all":
	default:
		scope = "history"
	}

	where := "s.user_id = $1"
	args := []any{userID}
	argN := 2
	switch scope {
	case "history":
		where += " AND s.folder_id IS NULL"
	case "saved":
		where += " AND s.folder_id IS NOT NULL"
	}
	if folderID != nil {
		where += fmt.Sprintf(" AND s.folder_id = $%d", argN)
		args = append(args, *folderID)
		argN++
	}
	args = append(args, limit)

	q := fmt.Sprintf(`
SELECT s.id, s.source_text, s.translation, s.source_lang, s.target_lang, s.folder_id,
       COALESCE(f.name, ''), s.created_at
FROM sentences s
LEFT JOIN folders f ON f.id = s.folder_id
WHERE %s
ORDER BY s.created_at DESC
LIMIT $%d`, where, argN)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Sentence
	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var sent Sentence
		if err := rows.Scan(&sent.ID, &sent.SourceText, &sent.Translation, &sent.SourceLang, &sent.TargetLang, &sent.FolderID, &sent.FolderName, &sent.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, sent)
		ids = append(ids, sent.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []Sentence{}, nil
	}

	bySentence, err := s.variantsFor(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Reformulations = bySentence[list[i].ID]["reformulation"]
		list[i].Related = bySentence[list[i].ID]["related"]
	}
	return list, nil
}

func (s *Store) GetSentence(ctx context.Context, userID string, id uuid.UUID) (Sentence, error) {
	const q = `
SELECT s.id, s.source_text, s.translation, s.source_lang, s.target_lang, s.folder_id,
       COALESCE(f.name, ''), s.created_at
FROM sentences s
LEFT JOIN folders f ON f.id = s.folder_id
WHERE s.id = $1 AND s.user_id = $2`
	var sent Sentence
	if err := s.pool.QueryRow(ctx, q, id, userID).Scan(&sent.ID, &sent.SourceText, &sent.Translation, &sent.SourceLang, &sent.TargetLang, &sent.FolderID, &sent.FolderName, &sent.CreatedAt); err != nil {
		return Sentence{}, err
	}
	bySentence, err := s.variantsFor(ctx, userID, []uuid.UUID{id})
	if err != nil {
		return Sentence{}, err
	}
	sent.Reformulations = bySentence[id]["reformulation"]
	sent.Related = bySentence[id]["related"]
	return sent, nil
}

func (s *Store) variantsFor(ctx context.Context, userID string, ids []uuid.UUID) (map[uuid.UUID]map[string][]Variant, error) {
	const q = `
SELECT id, sentence_id, kind, text
FROM variants WHERE user_id = $1 AND sentence_id = ANY($2)
ORDER BY created_at`
	rows, err := s.pool.Query(ctx, q, userID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[uuid.UUID]map[string][]Variant{}
	for rows.Next() {
		var v Variant
		var sid uuid.UUID
		if err := rows.Scan(&v.ID, &sid, &v.Kind, &v.Text); err != nil {
			return nil, err
		}
		if out[sid] == nil {
			out[sid] = map[string][]Variant{}
		}
		out[sid][v.Kind] = append(out[sid][v.Kind], v)
	}
	return out, rows.Err()
}

func (s *Store) InsertAudioJob(ctx context.Context, userID string, sentenceID, variantID *uuid.UUID, provider, model, language, text string) error {
	sum := sha256.Sum256([]byte(text))
	const q = `
INSERT INTO audio_jobs (id, user_id, sentence_id, variant_id, provider, model, language, text_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := s.pool.Exec(ctx, q, uuid.New(), userID, sentenceID, variantID, provider, model, language, hex.EncodeToString(sum[:]))
	return err
}

func (s *Store) GetUsage(ctx context.Context, userID string) (Usage, error) {
	const q = `SELECT translations, tts_seconds FROM usage_daily WHERE user_id = $1 AND day = CURRENT_DATE`
	var u Usage
	err := s.pool.QueryRow(ctx, q, userID).Scan(&u.Translations, &u.TTSSeconds)
	if err == pgx.ErrNoRows {
		return Usage{}, nil
	}
	return u, err
}

func (s *Store) IncrTranslations(ctx context.Context, userID string) error {
	const q = `
INSERT INTO usage_daily (user_id, day, translations, tts_seconds)
VALUES ($1, CURRENT_DATE, 1, 0)
ON CONFLICT (user_id, day) DO UPDATE SET translations = usage_daily.translations + 1`
	_, err := s.pool.Exec(ctx, q, userID)
	return err
}

func (s *Store) IncrTTSSeconds(ctx context.Context, userID string, seconds int) error {
	if seconds < 1 {
		seconds = 1
	}
	const q = `
INSERT INTO usage_daily (user_id, day, translations, tts_seconds)
VALUES ($1, CURRENT_DATE, 0, $2)
ON CONFLICT (user_id, day) DO UPDATE SET tts_seconds = usage_daily.tts_seconds + $2`
	_, err := s.pool.Exec(ctx, q, userID, seconds)
	return err
}

func (s *Store) ListFolders(ctx context.Context, userID string) ([]Folder, error) {
	const q = `SELECT id, name, created_at FROM folders WHERE user_id = $1 ORDER BY name`
	rows, err := s.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Folder
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.Name, &f.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	if list == nil {
		return []Folder{}, rows.Err()
	}
	return list, rows.Err()
}

func (s *Store) CreateFolder(ctx context.Context, userID, name string) (Folder, error) {
	const q = `
INSERT INTO folders (id, user_id, name)
VALUES ($1, $2, $3)
RETURNING id, name, created_at`
	var f Folder
	f.ID = uuid.New()
	err := s.pool.QueryRow(ctx, q, f.ID, userID, name).Scan(&f.ID, &f.Name, &f.CreatedAt)
	return f, err
}

func (s *Store) GetFolder(ctx context.Context, userID string, id uuid.UUID) (Folder, error) {
	const q = `SELECT id, name, created_at FROM folders WHERE id = $1 AND user_id = $2`
	var f Folder
	err := s.pool.QueryRow(ctx, q, id, userID).Scan(&f.ID, &f.Name, &f.CreatedAt)
	return f, err
}

func (s *Store) DeleteSentence(ctx context.Context, userID string, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM sentences WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ClearHistory(ctx context.Context, userID string) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM sentences WHERE user_id = $1 AND folder_id IS NULL`, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *Store) SetSentenceFolder(ctx context.Context, userID string, id uuid.UUID, folderID *uuid.UUID) (Sentence, error) {
	if folderID != nil {
		if _, err := s.GetFolder(ctx, userID, *folderID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Sentence{}, ErrNotFound
			}
			return Sentence{}, err
		}
	}
	const q = `
UPDATE sentences SET folder_id = $3
WHERE id = $1 AND user_id = $2`
	tag, err := s.pool.Exec(ctx, q, id, userID, folderID)
	if err != nil {
		return Sentence{}, err
	}
	if tag.RowsAffected() == 0 {
		return Sentence{}, ErrNotFound
	}
	return s.GetSentence(ctx, userID, id)
}
