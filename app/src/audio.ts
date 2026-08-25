import { Audio } from 'expo-av';
import { Directory, File, Paths } from 'expo-file-system';
import * as Speech from 'expo-speech';
import { Platform } from 'react-native';

import { fetchSpeech } from './api';
import type { User } from './types';

let current: Audio.Sound | null = null;

type CachedAudio = { uri: string; mime: string };
type CacheMeta = { ext: string; contentType: string };

const webCache = new Map<string, CachedAudio>();

export type SpeakOpts = {
  text: string;
  language: string;
  sentenceId?: string;
  variantId?: string;
  /** provider|model|voice — must match server TTS settings for cache hits */
  ttsTag?: string;
};

export function ttsCacheTag(user: Pick<User, 'tts_provider' | 'tts_model' | 'tts_voice'>): string {
  return `${user.tts_provider}|${user.tts_model}|${user.tts_voice ?? ''}`;
}

function voiceSuffix(ttsTag?: string): string {
  if (!ttsTag) return 'default';
  return hashText(ttsTag);
}

function cacheKey(opts: SpeakOpts): string {
  const voice = voiceSuffix(opts.ttsTag);
  if (opts.variantId) return `v-${opts.variantId}-${voice}`;
  if (opts.sentenceId) return `s-${opts.sentenceId}-${voice}`;
  return `t-${hashText(`${opts.language}:${opts.text}:${opts.ttsTag ?? ''}`)}`;
}

function hashText(text: string): string {
  let h = 5381;
  for (let i = 0; i < text.length; i++) {
    h = ((h << 5) + h) ^ text.charCodeAt(i);
  }
  return (h >>> 0).toString(36);
}

function sniffAudioFormat(data: ArrayBuffer): { mime: string; ext: string } | null {
  const u = new Uint8Array(data);
  if (u.length >= 4 && u[0] === 0x52 && u[1] === 0x49 && u[2] === 0x46 && u[3] === 0x46) {
    return { mime: 'audio/wav', ext: 'wav' };
  }
  if (u.length >= 3 && u[0] === 0x49 && u[1] === 0x44 && u[2] === 0x33) {
    return { mime: 'audio/mpeg', ext: 'mp3' };
  }
  if (u.length >= 2 && u[0] === 0xff && (u[1] & 0xe0) === 0xe0) {
    return { mime: 'audio/mpeg', ext: 'mp3' };
  }
  if (u.length >= 1 && (u[0] === 0x7b || u[0] === 0x5b)) {
    return null;
  }
  return { mime: 'audio/mpeg', ext: 'mp3' };
}

function audioDir(): Directory {
  const dir = new Directory(Paths.document, 'audio');
  dir.create({ intermediates: true, idempotent: true });
  return dir;
}

function invalidateCached(key: string) {
  if (Platform.OS === 'web') {
    const prev = webCache.get(key);
    if (prev?.uri.startsWith('blob:')) URL.revokeObjectURL(prev.uri);
    webCache.delete(key);
    return;
  }
  const dir = audioDir();
  for (const ext of ['mp3', 'wav', 'json']) {
    const file = new File(dir, `${key}.${ext}`);
    if (file.exists) file.delete();
  }
}

function loadCached(key: string): CachedAudio | null {
  if (Platform.OS === 'web') {
    return webCache.get(key) ?? null;
  }
  const dir = audioDir();
  const metaFile = new File(dir, `${key}.json`);
  if (!metaFile.exists) return null;
  let meta: CacheMeta;
  try {
    meta = JSON.parse(metaFile.textSync()) as CacheMeta;
  } catch {
    return null;
  }
  const audioFile = new File(dir, `${key}.${meta.ext}`);
  if (!audioFile.exists) return null;
  return { uri: audioFile.uri, mime: meta.contentType };
}

function saveCached(key: string, data: ArrayBuffer): CachedAudio {
  const sniffed = sniffAudioFormat(data);
  if (!sniffed) {
    throw new Error('Invalid audio received from server');
  }
  const { mime, ext } = sniffed;
  if (Platform.OS === 'web') {
    invalidateCached(key);
    const blob = new Blob([data], { type: mime });
    const uri = URL.createObjectURL(blob);
    const cached = { uri, mime };
    webCache.set(key, cached);
    return cached;
  }
  const dir = audioDir();
  const audioFile = new File(dir, `${key}.${ext}`);
  audioFile.create({ overwrite: true });
  audioFile.write(new Uint8Array(data));
  const metaFile = new File(dir, `${key}.json`);
  metaFile.create({ overwrite: true });
  metaFile.write(JSON.stringify({ ext, contentType: mime } satisfies CacheMeta));
  return { uri: audioFile.uri, mime };
}

async function stop() {
  Speech.stop();
  if (current) {
    try {
      await current.stopAsync();
      await current.unloadAsync();
    } catch {
      // ignore
    }
    current = null;
  }
}

async function playUri(uri: string): Promise<void> {
  await Audio.setAudioModeAsync({
    playsInSilentModeIOS: true,
    staysActiveInBackground: false,
    allowsRecordingIOS: false,
    shouldDuckAndroid: true,
    playThroughEarpieceAndroid: false,
  });
  const { sound } = await Audio.Sound.createAsync({ uri }, { shouldPlay: true });
  current = sound;
  sound.setOnPlaybackStatusUpdate((status) => {
    if (!status.isLoaded) return;
    if (status.didJustFinish) {
      void sound.unloadAsync().finally(() => {
        if (current === sound) current = null;
      });
    }
  });
}

function fallbackSpeak(text: string, language: string) {
  const lang = language === 'fr' ? 'fr-FR' : language === 'vi' ? 'vi-VN' : language;
  Speech.speak(text, { language: lang });
}

async function loadOrFetch(opts: SpeakOpts, key: string): Promise<CachedAudio> {
  const cached = loadCached(key);
  if (cached) return cached;

  try {
    const { data } = await fetchSpeech(opts);
    if (!sniffAudioFormat(data)) {
      throw new Error('Invalid audio received from server');
    }
    return saveCached(key, data);
  } catch (err) {
    console.error('tts fetch failed', err);
    fallbackSpeak(opts.text, opts.language);
    throw err instanceof Error ? err : new Error('Could not fetch speech');
  }
}

export async function speak(opts: SpeakOpts): Promise<void> {
  await stop();
  const key = cacheKey(opts);
  let cached = await loadOrFetch(opts, key);

  try {
    await playUri(cached.uri);
  } catch (err) {
    console.error('audio playback failed', { key, uri: cached.uri, err });
    invalidateCached(key);
    cached = await loadOrFetch(opts, key);
    try {
      await playUri(cached.uri);
    } catch (retryErr) {
      console.error('audio playback retry failed', retryErr);
      throw retryErr instanceof Error ? retryErr : new Error('Could not play audio');
    }
  }
}
