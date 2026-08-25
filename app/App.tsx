import { Ionicons } from '@expo/vector-icons';
import { StatusBar } from 'expo-status-bar';
import { useState } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';

import HistoryScreen from './src/screens/HistoryScreen';
import SettingsScreen from './src/screens/SettingsScreen';
import TranslateScreen from './src/screens/TranslateScreen';
import type { Sentence } from './src/types';

type Tab = 'translate' | 'history' | 'settings';

export default function App() {
  const [tab, setTab] = useState<Tab>('translate');
  const [historyKey, setHistoryKey] = useState(0);
  const [restore, setRestore] = useState<Sentence | null>(null);

  return (
    <SafeAreaProvider>
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right', 'bottom']}>
        <StatusBar style="dark" />
        <View style={styles.header}>
          <Text style={styles.brand}>Repetiter</Text>
        </View>
        <View style={styles.body}>
          <View style={{ flex: 1, display: tab === 'translate' ? 'flex' : 'none' }}>
            <TranslateScreen
              active={tab === 'translate'}
              restore={restore}
              onRestored={() => setRestore(null)}
              onTranslated={() => setHistoryKey((k) => k + 1)}
            />
          </View>
          <View style={{ flex: 1, display: tab === 'history' ? 'flex' : 'none' }}>
            <HistoryScreen
              active={tab === 'history'}
              refreshKey={historyKey}
              onOpen={(s) => {
                setRestore(s);
                setTab('translate');
              }}
            />
          </View>
          <View style={{ flex: 1, display: tab === 'settings' ? 'flex' : 'none' }}>
            <SettingsScreen active={tab === 'settings'} />
          </View>
        </View>
        <View style={styles.tabs}>
          <TabBtn icon="swap-horizontal" label="Translate" on={tab === 'translate'} onPress={() => setTab('translate')} />
          <TabBtn icon="time-outline" label="History" on={tab === 'history'} onPress={() => setTab('history')} />
          <TabBtn icon="settings-outline" label="Settings" on={tab === 'settings'} onPress={() => setTab('settings')} />
        </View>
      </SafeAreaView>
    </SafeAreaProvider>
  );
}

function TabBtn({
  icon,
  label,
  on,
  onPress,
}: {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  on: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable style={styles.tab} onPress={onPress}>
      <Ionicons name={icon} size={22} color={on ? '#1a73e8' : '#5f6368'} />
      <Text style={[styles.tabLabel, on && styles.tabLabelOn]}>{label}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: '#fff' },
  header: { paddingHorizontal: 20, paddingBottom: 8 },
  brand: { fontSize: 20, fontWeight: '700', color: '#1a73e8' },
  body: { flex: 1 },
  tabs: {
    flexDirection: 'row',
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: '#dadce0',
    paddingBottom: 8,
    paddingTop: 6,
  },
  tab: { flex: 1, alignItems: 'center', gap: 2 },
  tabLabel: { fontSize: 11, color: '#5f6368' },
  tabLabelOn: { color: '#1a73e8', fontWeight: '600' },
});
