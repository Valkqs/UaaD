import { beforeEach } from 'vitest';

function createStorageMock(): Storage {
  let store = new Map<string, string>();

  return {
    get length() {
      return store.size;
    },
    clear() {
      store = new Map<string, string>();
    },
    getItem(key: string) {
      return store.get(key) ?? null;
    },
    key(index: number) {
      return [...store.keys()][index] ?? null;
    },
    removeItem(key: string) {
      store.delete(key);
    },
    setItem(key: string, value: string) {
      store.set(key, String(value));
    },
  };
}

const storage = createStorageMock();

Object.defineProperty(window, 'localStorage', {
  configurable: true,
  value: storage,
});

Object.defineProperty(globalThis, 'localStorage', {
  configurable: true,
  value: storage,
});

beforeEach(() => {
  storage.clear();
});
