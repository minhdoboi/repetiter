import { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';

import { API_URL, getMe, listTTSVoices, patchMe } from '../api';
import type { TTSVoice, User } from '../types';

const TEXT_OPTIONS = [
  { provider: 'mock', model: 'mock', label: 'Mock (no API key)' },
  { provider: 'openai', model: 'gpt-4o-mini', label: 'OpenAI · gpt-4o-mini' },
  { provider: 'mistral', model: 'mistral-small-latest', label: 'Mistral · Small (EU)' },
  { provider: 'groq', model: 'openai/gpt-oss-20b', label: 'Groq · gpt-oss-20b' },
];

type TTSOption = {
  provider: string;
  model: string;
  label: string;
  presetVoices?: boolean;
  customVoice?: boolean;
};

const TTS_OPTIONS: TTSOption[] = [
  { provider: 'mock', model: 'mock', label: 'On-device voice (fallback)' },
  { provider: 'google', model: 'gtts', label: 'Google TTS (free)', presetVoices: true },
  { provider: 'openai', model: 'gpt-4o-mini-tts', label: 'OpenAI TTS', presetVoices: true },
  { provider: 'elevenlabs', model: 'eleven_flash_v2_5', label: 'ElevenLabs Flash', customVoice: true },
  { provider: 'mistral', model: 'voxtral-mini-tts-2603', label: 'Mistral Voxtral' },
  { provider: 'groq', model: 'canopylabs/orpheus-v1-english', label: 'Groq · Orpheus (EN)', presetVoices: true },
];

type Props = { active: boolean };

export default function SettingsScreen({ active }: Props) {
  const [user, setUser] = useState<User | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [customVoice, setCustomVoice] = useState('');
  const [voices, setVoices] = useState<TTSVoice[]>([]);
  const [voicesLoading, setVoicesLoading] = useState(false);

  useEffect(() => {
    if (!active) return;
    let cancelled = false;
    (async () => {
      try {
        const me = await getMe();
        if (!cancelled) {
          setUser(me);
          setCustomVoice(me.tts_voice ?? '');
        }
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Could not load settings');
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [active]);

  const activeTTS = TTS_OPTIONS.find(
    (opt) => user?.tts_provider === opt.provider && user?.tts_model === opt.model,
  );

  useEffect(() => {
    if (!active || !activeTTS) {
      setVoices([]);
      return;
    }
    if (activeTTS.customVoice) {
      setVoices([]);
      return;
    }
    let cancelled = false;
    setVoicesLoading(true);
    setError(null);
    (async () => {
      try {
        const listed = await listTTSVoices(activeTTS.provider);
        if (!cancelled) setVoices(listed);
      } catch (e) {
        if (!cancelled) {
          setVoices([]);
          setError(e instanceof Error ? e.message : 'Could not load voices');
        }
      } finally {
        if (!cancelled) setVoicesLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [active, activeTTS?.provider, activeTTS?.model, activeTTS?.customVoice]);

  const save = async (patch: Partial<User>) => {
    setSaving(true);
    setError(null);
    try {
      const updated = await patchMe(patch);
      setUser(updated);
      if (patch.tts_voice !== undefined) {
        setCustomVoice(updated.tts_voice ?? '');
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not save');
    } finally {
      setSaving(false);
    }
  };

  if (!user && !error) {
    return (
      <View style={styles.center}>
        <ActivityIndicator color="#1a73e8" />
      </View>
    );
  }

  return (
    <ScrollView style={styles.screen} contentContainerStyle={styles.content}>
      <Text style={styles.hint}>API {API_URL}</Text>
      {error ? <Text style={styles.error}>{error}</Text> : null}

      <Text style={styles.section}>Text</Text>
      {TEXT_OPTIONS.map((opt) => {
        const on = user?.text_provider === opt.provider && user?.text_model === opt.model;
        return (
          <Pressable
            key={opt.label}
            style={[styles.option, on && styles.optionOn]}
            onPress={() => save({ text_provider: opt.provider, text_model: opt.model })}
          >
            <Text style={[styles.optionText, on && styles.optionTextOn]}>{opt.label}</Text>
          </Pressable>
        );
      })}

      <Text style={styles.section}>Voice provider</Text>
      {TTS_OPTIONS.map((opt) => {
        const on = user?.tts_provider === opt.provider && user?.tts_model === opt.model;
        return (
          <Pressable
            key={opt.label}
            style={[styles.option, on && styles.optionOn]}
            onPress={() => save({ tts_provider: opt.provider, tts_model: opt.model })}
          >
            <Text style={[styles.optionText, on && styles.optionTextOn]}>{opt.label}</Text>
          </Pressable>
        );
      })}

      {activeTTS && !activeTTS.customVoice ? (
        <>
          <Text style={styles.section}>Voice</Text>
          {voicesLoading ? <ActivityIndicator color="#1a73e8" style={{ marginBottom: 12 }} /> : null}
          {!voicesLoading && voices.length === 0 ? (
            <Text style={styles.hint}>No voices available for this provider.</Text>
          ) : null}
          {voices.map((voice) => {
            const on = user?.tts_voice === voice.id;
            const suffix = voice.languages?.length ? ` · ${voice.languages.join(', ')}` : '';
            return (
              <Pressable
                key={voice.id}
                style={[styles.option, on && styles.optionOn]}
                onPress={() => save({ tts_voice: voice.id })}
              >
                <Text style={[styles.optionText, on && styles.optionTextOn]}>
                  {voice.label}
                  {suffix}
                </Text>
              </Pressable>
            );
          })}
        </>
      ) : null}

      {activeTTS?.customVoice ? (
        <>
          <Text style={styles.section}>ElevenLabs voice id</Text>
          <TextInput
            style={styles.input}
            value={customVoice}
            onChangeText={setCustomVoice}
            placeholder="Voice id from elevenlabs.io"
            autoCapitalize="none"
            autoCorrect={false}
          />
          <Pressable
            style={[styles.option, styles.saveVoice]}
            onPress={() => save({ tts_voice: customVoice.trim() })}
          >
            <Text style={styles.optionText}>Save voice id</Text>
          </Pressable>
        </>
      ) : null}

      {saving ? <ActivityIndicator style={{ marginTop: 16 }} color="#1a73e8" /> : null}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#fff' },
  content: { padding: 20, paddingBottom: 40 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#fff' },
  hint: { color: '#80868b', fontSize: 12, marginBottom: 16 },
  error: { color: '#d93025', marginBottom: 12 },
  section: {
    fontSize: 12,
    fontWeight: '600',
    color: '#5f6368',
    letterSpacing: 0.4,
    textTransform: 'uppercase',
    marginTop: 16,
    marginBottom: 8,
  },
  option: {
    paddingVertical: 12,
    paddingHorizontal: 14,
    borderRadius: 8,
    backgroundColor: '#f1f3f4',
    marginBottom: 8,
  },
  optionOn: { backgroundColor: '#e8f0fe' },
  optionText: { fontSize: 15, color: '#202124' },
  optionTextOn: { color: '#1a73e8', fontWeight: '600' },
  input: {
    borderWidth: 1,
    borderColor: '#dadce0',
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 15,
    marginBottom: 8,
  },
  saveVoice: { alignItems: 'center' },
});
