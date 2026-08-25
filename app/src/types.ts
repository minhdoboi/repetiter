export type Lang = 'fr' | 'vi';

export type Variant = {
  id: string;
  kind: 'reformulation' | 'related';
  text: string;
};

export type Sentence = {
  id: string;
  source_text: string;
  translation: string;
  source_lang: Lang;
  target_lang: Lang;
  created_at: string;
  folder_id?: string;
  folder_name?: string;
  reformulations?: Variant[];
  related?: Variant[];
};

export type Folder = {
  id: string;
  name: string;
  created_at: string;
};

export type TTSVoice = {
  id: string;
  label: string;
  languages?: string[];
};

export type User = {
  id: string;
  source_lang: Lang;
  target_lang: Lang;
  text_provider: string;
  text_model: string;
  tts_provider: string;
  tts_model: string;
  tts_voice?: string;
  plan: string;
};

export const LANGS: { code: Lang; label: string }[] = [
  { code: 'fr', label: 'Français' },
  { code: 'vi', label: 'Tiếng Việt' },
];

export function langLabel(code: string): string {
  return LANGS.find((l) => l.code === code)?.label ?? code;
}
