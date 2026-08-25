import { Ionicons } from '@expo/vector-icons';
import { useEffect, useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Linking,
  Modal,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  useWindowDimensions,
  View,
} from 'react-native';

import { createSentence, getMe } from '../api';
import { speak, ttsCacheTag } from '../audio';
import { LANGS, langLabel, type Lang, type Sentence, type Variant } from '../types';

type Props = {
  active: boolean;
  restore?: Sentence | null;
  onRestored?: () => void;
  onTranslated?: (s: Sentence) => void;
};

export default function TranslateScreen({ active, restore, onRestored, onTranslated }: Props) {
  const { width } = useWindowDimensions();
  const sideBySide = width >= 800;

  const [sourceLang, setSourceLang] = useState<Lang>('fr');
  const [targetLang, setTargetLang] = useState<Lang>('vi');
  const [source, setSource] = useState('');
  const [sentence, setSentence] = useState<Sentence | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [picker, setPicker] = useState<'source' | 'target' | null>(null);
  const [speaking, setSpeaking] = useState<string | null>(null);
  const [ttsTag, setTtsTag] = useState('');

  useEffect(() => {
    if (!active) return;
    let cancelled = false;
    (async () => {
      try {
        const me = await getMe();
        if (!cancelled) setTtsTag(ttsCacheTag(me));
      } catch {
        // keep previous tag; server still uses saved settings
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [active]);

  useEffect(() => {
    if (!restore) return;
    setSourceLang(restore.source_lang);
    setTargetLang(restore.target_lang);
    setSource(restore.source_text);
    setSentence(restore);
    setError(null);
    onRestored?.();
  }, [restore, onRestored]);

  const translation = sentence?.translation ?? '';

  const swap = () => {
    setSourceLang(targetLang);
    setTargetLang(sourceLang);
    setSource(translation);
    setSentence((prev) =>
      prev
        ? {
            ...prev,
            source_text: prev.translation,
            translation: source,
            source_lang: prev.target_lang,
            target_lang: prev.source_lang,
            reformulations: [],
            related: [],
          }
        : null,
    );
  };

  const pickLang = (side: 'source' | 'target', code: Lang) => {
    if (side === 'source') {
      if (code === targetLang) swap();
      else setSourceLang(code);
    } else if (code === sourceLang) swap();
    else setTargetLang(code);
    setPicker(null);
  };

  const translate = async () => {
    const text = source.trim();
    if (!text || loading) return;
    setLoading(true);
    setError(null);
    try {
      const result = await createSentence(text, sourceLang, targetLang);
      setSentence(result);
      onTranslated?.(result);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Translation failed');
    } finally {
      setLoading(false);
    }
  };

  const play = async (text: string, language: string, key: string, sentenceId?: string, variantId?: string) => {
    setSpeaking(key);
    setError(null);
    try {
      await speak({ text, language, sentenceId, variantId, ttsTag });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not play audio');
    } finally {
      setSpeaking(null);
    }
  };

  const useSuggestion = (v: Variant) => {
    setSource(v.text);
    setSourceLang(targetLang);
    setTargetLang(sourceLang);
    setSentence(null);
  };

  const sourcePane = (
    <View style={[styles.pane, sideBySide && styles.paneWide]}>
      <TextInput
        style={styles.input}
        multiline
        placeholder={sourceLang === 'fr' ? 'Saisissez du texte' : 'Nhập văn bản'}
        placeholderTextColor="#80868b"
        value={source}
        onChangeText={(t) => {
          setSource(t);
          if (sentence) setSentence(null);
        }}
        textAlignVertical="top"
        autoCorrect
      />
      <View style={styles.paneFooter}>
        {source.length > 0 ? (
          <Pressable onPress={() => { setSource(''); setSentence(null); }} hitSlop={8}>
            <Ionicons name="close" size={22} color="#5f6368" />
          </Pressable>
        ) : (
          <View />
        )}
        <Text style={styles.count}>{source.length}</Text>
      </View>
    </View>
  );

  const targetPane = (
    <View style={[styles.pane, styles.targetPane, sideBySide && styles.paneWide]}>
      {loading ? (
        <View style={styles.loadingBox}>
          <ActivityIndicator color="#1a73e8" />
        </View>
      ) : (
        <Text style={[styles.output, !translation && styles.outputEmpty]}>
          {translation || (targetLang === 'vi' ? 'Bản dịch' : 'Traduction')}
        </Text>
      )}
      {translation ? (
        <View style={styles.paneFooter}>
          <Pressable
            onPress={() => play(translation, targetLang, 'main', sentence?.id)}
            disabled={speaking === 'main'}
            hitSlop={8}
          >
            <Ionicons
              name={speaking === 'main' ? 'volume-high' : 'volume-medium-outline'}
              size={22}
              color="#1a73e8"
            />
          </Pressable>
        </View>
      ) : null}
    </View>
  );

  const suggestions = useMemo(() => {
    if (!sentence) return null;
    const reform = sentence.reformulations ?? [];
    const related = sentence.related ?? [];
    if (reform.length === 0 && related.length === 0) return null;
    return (
      <View style={styles.suggestions}>
        {reform.length > 0 ? (
          <SuggestionGroup
            title="Reformulations"
            items={reform}
            lang={targetLang}
            speaking={speaking}
            onPlay={play}
            onUse={useSuggestion}
            googleTranslateTo={sourceLang}
          />
        ) : null}
        {related.length > 0 ? (
          <SuggestionGroup
            title="Related"
            items={related}
            lang={targetLang}
            speaking={speaking}
            onPlay={play}
            onUse={useSuggestion}
            googleTranslateTo={sourceLang}
          />
        ) : null}
      </View>
    );
  }, [sentence, speaking, targetLang, sourceLang]);

  return (
    <View style={styles.screen}>
      <View style={styles.langBar}>
        <Pressable style={styles.langBtn} onPress={() => setPicker('source')}>
          <Text style={styles.langText}>{langLabel(sourceLang)}</Text>
          <Ionicons name="chevron-down" size={16} color="#1a73e8" />
        </Pressable>
        <Pressable onPress={swap} style={styles.swap} hitSlop={12}>
          <Ionicons name="swap-horizontal" size={22} color="#5f6368" />
        </Pressable>
        <Pressable style={styles.langBtn} onPress={() => setPicker('target')}>
          <Text style={styles.langText}>{langLabel(targetLang)}</Text>
          <Ionicons name="chevron-down" size={16} color="#1a73e8" />
        </Pressable>
      </View>

      <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
        <View style={[styles.panes, sideBySide && styles.panesRow]}>
          {sourcePane}
          <View style={sideBySide ? styles.vDivider : styles.hDivider} />
          <View style={sideBySide ? styles.rightCol : undefined}>
            {targetPane}
            {sideBySide ? suggestions : null}
          </View>
        </View>
        {!sideBySide ? suggestions : null}

        {error ? <Text style={styles.error}>{error}</Text> : null}

        <Pressable
          style={[styles.translateBtn, (!source.trim() || loading) && styles.translateBtnOff]}
          onPress={translate}
          disabled={!source.trim() || loading}
        >
          <Text style={styles.translateLabel}>{loading ? '…' : 'Translate'}</Text>
        </Pressable>
      </ScrollView>

      <Modal visible={picker !== null} transparent animationType="fade" onRequestClose={() => setPicker(null)}>
        <Pressable style={styles.modalBg} onPress={() => setPicker(null)}>
          <View style={styles.modalCard}>
            <Text style={styles.modalTitle}>{picker === 'source' ? 'Translate from' : 'Translate to'}</Text>
            {LANGS.map((l) => (
              <Pressable key={l.code} style={styles.modalRow} onPress={() => picker && pickLang(picker, l.code)}>
                <Text style={styles.modalRowText}>{l.label}</Text>
                {(picker === 'source' ? sourceLang : targetLang) === l.code ? (
                  <Ionicons name="checkmark" size={20} color="#1a73e8" />
                ) : null}
              </Pressable>
            ))}
          </View>
        </Pressable>
      </Modal>
    </View>
  );
}

function googleTranslateUrl(text: string, fromLang: string, toLang: string): string {
  const q = new URLSearchParams({
    sl: fromLang,
    tl: toLang,
    text,
    op: 'translate',
  });
  return `https://translate.google.com/?${q.toString()}`;
}

function SuggestionGroup({
  title,
  items,
  lang,
  speaking,
  onPlay,
  onUse,
  googleTranslateTo,
}: {
  title: string;
  items: Variant[];
  lang: string;
  speaking: string | null;
  onPlay: (text: string, language: string, key: string, sentenceId?: string, variantId?: string) => void;
  onUse: (v: Variant) => void;
  googleTranslateTo?: string;
}) {
  return (
    <View style={styles.group}>
      <Text style={styles.groupTitle}>{title}</Text>
      {items.map((v) => (
        <View key={v.id} style={styles.chip}>
          <Pressable style={styles.chipTextWrap} onPress={() => onUse(v)}>
            <Text style={styles.chipText}>{v.text}</Text>
          </Pressable>
          {googleTranslateTo ? (
            <Pressable
              onPress={() => Linking.openURL(googleTranslateUrl(v.text, lang, googleTranslateTo))}
              hitSlop={8}
              accessibilityLabel="Open in Google Translate"
            >
              <Ionicons name="open-outline" size={18} color="#1a73e8" />
            </Pressable>
          ) : null}
          <Pressable onPress={() => onPlay(v.text, lang, v.id, undefined, v.id)} hitSlop={8}>
            <Ionicons
              name={speaking === v.id ? 'volume-high' : 'volume-medium-outline'}
              size={18}
              color="#1a73e8"
            />
          </Pressable>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#fff' },
  langBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 8,
    paddingVertical: 10,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#dadce0',
  },
  langBtn: { flex: 1, flexDirection: 'row', alignItems: 'center', justifyContent: 'center', gap: 4, padding: 8 },
  langText: { color: '#1a73e8', fontSize: 15, fontWeight: '600' },
  swap: { padding: 8 },
  scroll: { paddingBottom: 32 },
  panes: { backgroundColor: '#fff' },
  panesRow: { flexDirection: 'row', alignItems: 'stretch', minHeight: 280 },
  pane: { minHeight: 160, paddingHorizontal: 20, paddingTop: 16, paddingBottom: 8 },
  paneWide: { flex: 1, minHeight: 240 },
  targetPane: { backgroundColor: '#fff' },
  rightCol: { flex: 1 },
  input: { fontSize: 22, lineHeight: 30, color: '#202124', minHeight: 120, flex: 1 },
  output: { fontSize: 22, lineHeight: 30, color: '#202124', minHeight: 120 },
  outputEmpty: { color: '#80868b' },
  loadingBox: { minHeight: 120, justifyContent: 'center' },
  paneFooter: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginTop: 8 },
  count: { color: '#80868b', fontSize: 12 },
  hDivider: { height: StyleSheet.hairlineWidth, backgroundColor: '#dadce0', marginHorizontal: 12 },
  vDivider: { width: StyleSheet.hairlineWidth, backgroundColor: '#dadce0' },
  suggestions: { paddingHorizontal: 16, paddingTop: 8, paddingBottom: 8 },
  group: { marginBottom: 16 },
  groupTitle: { fontSize: 12, fontWeight: '600', color: '#5f6368', letterSpacing: 0.4, marginBottom: 8, textTransform: 'uppercase' },
  chip: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#f1f3f4',
    borderRadius: 8,
    paddingVertical: 10,
    paddingHorizontal: 12,
    marginBottom: 8,
    gap: 10,
  },
  chipTextWrap: { flex: 1 },
  chipText: { fontSize: 16, color: '#202124', lineHeight: 22 },
  error: { color: '#d93025', paddingHorizontal: 20, paddingTop: 12 },
  translateBtn: {
    alignSelf: 'center',
    marginTop: 16,
    backgroundColor: '#1a73e8',
    paddingHorizontal: 28,
    paddingVertical: 12,
    borderRadius: 24,
  },
  translateBtnOff: { opacity: 0.45 },
  translateLabel: { color: '#fff', fontSize: 15, fontWeight: '600' },
  modalBg: { flex: 1, backgroundColor: 'rgba(32,33,36,0.4)', justifyContent: 'center', padding: 24 },
  modalCard: { backgroundColor: '#fff', borderRadius: 8, paddingVertical: 8 },
  modalTitle: { fontSize: 16, fontWeight: '600', paddingHorizontal: 16, paddingVertical: 12, color: '#202124' },
  modalRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingHorizontal: 16, paddingVertical: 14 },
  modalRowText: { fontSize: 16, color: '#202124' },
});
