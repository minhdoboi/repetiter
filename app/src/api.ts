import { Platform } from 'react-native';

import type { Folder, Lang, Sentence, TTSVoice, User } from './types';

const DEFAULT_URL = Platform.OS === 'android' ? 'http://10.0.2.2:8080' : 'http://localhost:8080';

export const API_URL = (process.env.EXPO_PUBLIC_API_URL ?? DEFAULT_URL).replace(/\/$/, '');

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const hasBody = init.body != null && init.body !== '';
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(hasBody ? { 'Content-Type': 'application/json' } : {}),
      ...(init.headers ?? {}),
    },
  });
  const text = await res.text();
  if (!res.ok) {
    let msg = res.statusText;
    try {
      msg = (JSON.parse(text) as { error?: string }).error || msg;
    } catch {
      if (text) msg = text.slice(0, 200);
    }
    throw new Error(msg);
  }
  if (res.status === 204 || !text) {
    return {} as T;
  }
  return JSON.parse(text) as T;
}

export function getMe() {
  return request<User>('/v1/me');
}

export function patchMe(body: Partial<User>) {
  return request<User>('/v1/me', { method: 'PATCH', body: JSON.stringify(body) });
}

export function createSentence(sourceText: string, sourceLang: Lang, targetLang: Lang) {
  return request<Sentence>('/v1/sentences', {
    method: 'POST',
    body: JSON.stringify({
      source_text: sourceText,
      source_lang: sourceLang,
      target_lang: targetLang,
    }),
  });
}

export function listSentences(opts?: { scope?: 'history' | 'saved'; folderId?: string }) {
  const params = new URLSearchParams();
  if (opts?.scope) params.set('scope', opts.scope);
  if (opts?.folderId) params.set('folder_id', opts.folderId);
  const q = params.toString();
  return request<{ sentences: Sentence[] }>(`/v1/sentences${q ? `?${q}` : ''}`).then((r) => r.sentences ?? []);
}

export function listFolders() {
  return request<{ folders: Folder[] }>('/v1/folders').then((r) => r.folders ?? []);
}

export function createFolder(name: string) {
  return request<Folder>('/v1/folders', { method: 'POST', body: JSON.stringify({ name }) });
}

export function deleteSentence(id: string) {
  return request<void>(`/v1/sentences/${id}`, { method: 'DELETE' });
}

export function clearHistory() {
  return request<{ deleted: number }>('/v1/sentences/history', { method: 'DELETE' });
}

export function moveSentenceToFolder(id: string, folderId: string | null) {
  return request<Sentence>(`/v1/sentences/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ folder_id: folderId }),
  });
}

export function listTTSVoices(provider: string) {
  return request<{ voices: TTSVoice[] }>(`/v1/tts/voices?provider=${encodeURIComponent(provider)}`).then(
    (r) => r.voices ?? [],
  );
}

export async function fetchSpeech(opts: {
  text: string;
  language: string;
  sentenceId?: string;
  variantId?: string;
}): Promise<{ data: ArrayBuffer; contentType: string }> {
  const res = await fetch(`${API_URL}/v1/tts`, {
    method: 'POST',
    headers: { Accept: 'audio/mpeg, audio/wav, audio/*', 'Content-Type': 'application/json' },
    body: JSON.stringify({
      text: opts.text,
      language: opts.language,
      sentence_id: opts.sentenceId ?? '',
      variant_id: opts.variantId ?? '',
    }),
  });
  if (!res.ok) {
    const text = await res.text();
    let msg = res.statusText;
    try {
      msg = (JSON.parse(text) as { error?: string }).error || msg;
    } catch {
      if (text) msg = text.slice(0, 200);
    }
    throw new Error(msg);
  }
  const contentType = res.headers.get('Content-Type') || 'audio/mpeg';
  return { data: await res.arrayBuffer(), contentType };
}
