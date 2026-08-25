import { Ionicons } from '@expo/vector-icons';
import { useCallback, useEffect, useState } from 'react';
import {
  ActivityIndicator,
  FlatList,
  Modal,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';

import {
  clearHistory,
  createFolder,
  deleteSentence,
  listFolders,
  listSentences,
  moveSentenceToFolder,
} from '../api';
import { langLabel, type Folder, type Sentence } from '../types';

type Props = {
  active: boolean;
  onOpen: (s: Sentence) => void;
  refreshKey: number;
};

type Scope = 'history' | 'saved';

export default function HistoryScreen({ active, onOpen, refreshKey }: Props) {
  const [scope, setScope] = useState<Scope>('history');
  const [items, setItems] = useState<Sentence[] | null>(null);
  const [folders, setFolders] = useState<Folder[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [saveTarget, setSaveTarget] = useState<Sentence | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Sentence | null>(null);
  const [confirmClearOpen, setConfirmClearOpen] = useState(false);
  const [newFolderName, setNewFolderName] = useState('');
  const [selectedFolderId, setSelectedFolderId] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const [list, flds] = await Promise.all([
        listSentences({ scope }),
        listFolders(),
      ]);
      setItems(list);
      setFolders(flds);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load history');
    }
  }, [scope]);

  useEffect(() => {
    if (!active) return;
    let cancelled = false;
    setItems(null);
    setError(null);
    (async () => {
      try {
        await load();
      } catch {
        // load sets error
      }
      if (cancelled) return;
    })();
    return () => {
      cancelled = true;
    };
  }, [active, refreshKey, load]);

  const runDelete = async (item: Sentence) => {
    setBusy(true);
    setError(null);
    try {
      await deleteSentence(item.id);
      setItems((prev) => prev?.filter((s) => s.id !== item.id) ?? []);
      setDeleteTarget(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not delete');
    } finally {
      setBusy(false);
    }
  };

  const runClear = async () => {
    setBusy(true);
    setError(null);
    try {
      await clearHistory();
      setItems([]);
      setConfirmClearOpen(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not clear history');
    } finally {
      setBusy(false);
    }
  };

  const openSaveModal = async (item: Sentence) => {
    setSaveTarget(item);
    setNewFolderName('');
    setSelectedFolderId(folders[0]?.id ?? null);
    try {
      const flds = await listFolders();
      setFolders(flds);
      setSelectedFolderId(flds[0]?.id ?? null);
    } catch {
      // use cached folders
    }
  };

  const saveToFolder = async () => {
    if (!saveTarget) return;
    setBusy(true);
    setError(null);
    try {
      let folderId = selectedFolderId;
      const name = newFolderName.trim();
      if (name) {
        const folder = await createFolder(name);
        folderId = folder.id;
        setFolders((prev) => [...prev, folder].sort((a, b) => a.name.localeCompare(b.name)));
      }
      if (!folderId) {
        setError('Pick or create a folder');
        return;
      }
      await moveSentenceToFolder(saveTarget.id, folderId);
      setSaveTarget(null);
      if (scope === 'history') {
        setItems((prev) => prev?.filter((s) => s.id !== saveTarget.id) ?? []);
      } else {
        await load();
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not save to folder');
    } finally {
      setBusy(false);
    }
  };

  if (items === null && !error) {
    return (
      <View style={styles.center}>
        <ActivityIndicator color="#1a73e8" />
      </View>
    );
  }

  return (
    <View style={styles.screen}>
      <View style={styles.toolbar}>
        <View style={styles.segments}>
          <Pressable
            style={[styles.segment, scope === 'history' && styles.segmentOn]}
            onPress={() => setScope('history')}
          >
            <Text style={[styles.segmentText, scope === 'history' && styles.segmentTextOn]}>Recent</Text>
          </Pressable>
          <Pressable
            style={[styles.segment, scope === 'saved' && styles.segmentOn]}
            onPress={() => setScope('saved')}
          >
            <Text style={[styles.segmentText, scope === 'saved' && styles.segmentTextOn]}>Saved</Text>
          </Pressable>
        </View>
        {scope === 'history' && (items?.length ?? 0) > 0 ? (
          <Pressable onPress={() => setConfirmClearOpen(true)} disabled={busy} hitSlop={8}>
            <Text style={styles.clearBtn}>Clear all</Text>
          </Pressable>
        ) : null}
      </View>

      <FlatList
        style={styles.list}
        data={items ?? []}
        keyExtractor={(s) => s.id}
        contentContainerStyle={styles.content}
        ListEmptyComponent={
          <Text style={styles.empty}>
            {scope === 'history'
              ? 'No sentences yet. Translate one to start history.'
              : 'Nothing saved yet. Use the folder icon on a recent entry to keep it.'}
          </Text>
        }
        ListHeaderComponent={error ? <Text style={styles.error}>{error}</Text> : null}
        renderItem={({ item }) => (
          <View style={styles.row}>
            <Pressable style={styles.rowMain} onPress={() => onOpen(item)}>
              {item.folder_name ? <Text style={styles.folderTag}>{item.folder_name}</Text> : null}
              <Text style={styles.langs}>
                {langLabel(item.source_lang)} → {langLabel(item.target_lang)}
              </Text>
              <Text style={styles.source} numberOfLines={2}>
                {item.source_text}
              </Text>
              <Text style={styles.target} numberOfLines={2}>
                {item.translation}
              </Text>
            </Pressable>
            <View style={styles.actions}>
              {scope === 'history' ? (
                <Pressable onPress={() => openSaveModal(item)} hitSlop={8} disabled={busy}>
                  <Ionicons name="folder-outline" size={22} color="#1a73e8" />
                </Pressable>
              ) : null}
              <Pressable onPress={() => setDeleteTarget(item)} hitSlop={8} disabled={busy}>
                <Ionicons name="trash-outline" size={22} color="#d93025" />
              </Pressable>
            </View>
          </View>
        )}
      />

      <Modal visible={saveTarget !== null} transparent animationType="fade" onRequestClose={() => setSaveTarget(null)}>
        <Pressable style={styles.modalBg} onPress={() => setSaveTarget(null)}>
          <Pressable style={styles.modalCard} onPress={(e) => e.stopPropagation()}>
            <Text style={styles.modalTitle}>Save to folder</Text>
            {folders.map((f) => {
              const on = selectedFolderId === f.id && !newFolderName.trim();
              return (
                <Pressable
                  key={f.id}
                  style={[styles.folderRow, on && styles.folderRowOn]}
                  onPress={() => {
                    setSelectedFolderId(f.id);
                    setNewFolderName('');
                  }}
                >
                  <Ionicons name="folder" size={18} color={on ? '#1a73e8' : '#5f6368'} />
                  <Text style={[styles.folderRowText, on && styles.folderRowTextOn]}>{f.name}</Text>
                </Pressable>
              );
            })}
            <TextInput
              style={styles.input}
              value={newFolderName}
              onChangeText={(t) => {
                setNewFolderName(t);
                if (t.trim()) setSelectedFolderId(null);
              }}
              placeholder="Or create a new folder"
              autoCapitalize="sentences"
            />
            <Pressable style={[styles.saveBtn, busy && styles.saveBtnOff]} onPress={saveToFolder} disabled={busy}>
              <Text style={styles.saveBtnText}>Save</Text>
            </Pressable>
          </Pressable>
        </Pressable>
      </Modal>

      <Modal visible={deleteTarget !== null} transparent animationType="fade" onRequestClose={() => setDeleteTarget(null)}>
        <Pressable style={styles.modalBg} onPress={() => setDeleteTarget(null)}>
          <Pressable style={styles.modalCard} onPress={(e) => e.stopPropagation()}>
            <Text style={styles.modalTitle}>Delete sentence?</Text>
            <Text style={styles.modalBody}>This entry will be removed permanently.</Text>
            <View style={styles.confirmActions}>
              <Pressable style={styles.cancelBtn} onPress={() => setDeleteTarget(null)} disabled={busy}>
                <Text style={styles.cancelBtnText}>Cancel</Text>
              </Pressable>
              <Pressable
                style={[styles.deleteBtn, busy && styles.saveBtnOff]}
                onPress={() => deleteTarget && runDelete(deleteTarget)}
                disabled={busy}
              >
                <Text style={styles.deleteBtnText}>Delete</Text>
              </Pressable>
            </View>
          </Pressable>
        </Pressable>
      </Modal>

      <Modal visible={confirmClearOpen} transparent animationType="fade" onRequestClose={() => setConfirmClearOpen(false)}>
        <Pressable style={styles.modalBg} onPress={() => setConfirmClearOpen(false)}>
          <Pressable style={styles.modalCard} onPress={(e) => e.stopPropagation()}>
            <Text style={styles.modalTitle}>Clear history?</Text>
            <Text style={styles.modalBody}>Remove all recent entries. Saved items in folders are kept.</Text>
            <View style={styles.confirmActions}>
              <Pressable style={styles.cancelBtn} onPress={() => setConfirmClearOpen(false)} disabled={busy}>
                <Text style={styles.cancelBtnText}>Cancel</Text>
              </Pressable>
              <Pressable style={[styles.deleteBtn, busy && styles.saveBtnOff]} onPress={runClear} disabled={busy}>
                <Text style={styles.deleteBtnText}>Clear all</Text>
              </Pressable>
            </View>
          </Pressable>
        </Pressable>
      </Modal>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#fff' },
  list: { flex: 1 },
  content: { paddingBottom: 32 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#fff' },
  toolbar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 10,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#dadce0',
  },
  segments: { flexDirection: 'row', backgroundColor: '#f1f3f4', borderRadius: 8, padding: 2 },
  segment: { paddingHorizontal: 14, paddingVertical: 6, borderRadius: 6 },
  segmentOn: { backgroundColor: '#fff' },
  segmentText: { fontSize: 14, color: '#5f6368' },
  segmentTextOn: { color: '#1a73e8', fontWeight: '600' },
  clearBtn: { fontSize: 14, color: '#d93025', fontWeight: '600' },
  empty: { color: '#5f6368', padding: 24, textAlign: 'center' },
  error: { color: '#d93025', padding: 16 },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#dadce0',
  },
  rowMain: { flex: 1, paddingHorizontal: 20, paddingVertical: 14 },
  actions: { flexDirection: 'row', gap: 16, paddingRight: 16 },
  folderTag: { fontSize: 11, color: '#5f6368', marginBottom: 4, textTransform: 'uppercase', letterSpacing: 0.3 },
  langs: { fontSize: 12, color: '#1a73e8', fontWeight: '600', marginBottom: 4 },
  source: { fontSize: 16, color: '#202124', marginBottom: 4 },
  target: { fontSize: 15, color: '#5f6368' },
  modalBg: { flex: 1, backgroundColor: 'rgba(32,33,36,0.4)', justifyContent: 'center', padding: 24 },
  modalCard: { backgroundColor: '#fff', borderRadius: 8, padding: 16 },
  modalTitle: { fontSize: 16, fontWeight: '600', marginBottom: 12, color: '#202124' },
  modalBody: { fontSize: 15, color: '#5f6368', marginBottom: 16, lineHeight: 22 },
  confirmActions: { flexDirection: 'row', justifyContent: 'flex-end', gap: 12 },
  cancelBtn: { paddingVertical: 10, paddingHorizontal: 12 },
  cancelBtnText: { fontSize: 15, color: '#5f6368' },
  deleteBtn: {
    backgroundColor: '#d93025',
    borderRadius: 8,
    paddingVertical: 10,
    paddingHorizontal: 16,
  },
  deleteBtnText: { color: '#fff', fontWeight: '600', fontSize: 15 },
  folderRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    paddingVertical: 10,
    paddingHorizontal: 8,
    borderRadius: 6,
    marginBottom: 4,
  },
  folderRowOn: { backgroundColor: '#e8f0fe' },
  folderRowText: { fontSize: 15, color: '#202124' },
  folderRowTextOn: { color: '#1a73e8', fontWeight: '600' },
  input: {
    borderWidth: 1,
    borderColor: '#dadce0',
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 15,
    marginTop: 8,
    marginBottom: 12,
  },
  saveBtn: {
    backgroundColor: '#1a73e8',
    borderRadius: 8,
    paddingVertical: 12,
    alignItems: 'center',
  },
  saveBtnOff: { opacity: 0.5 },
  saveBtnText: { color: '#fff', fontWeight: '600', fontSize: 15 },
});
