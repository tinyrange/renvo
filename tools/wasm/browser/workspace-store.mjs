const DATABASE = "renvo.web.ide";
const VERSION = 1;

export async function loadCurrentProject() {
  const db = await openProjectDatabase();
  return request(db.transaction("projects", "readonly").objectStore("projects").get("current"));
}

export async function saveCurrentProject(project) {
  const db = await openProjectDatabase();
  await request(db.transaction("projects", "readwrite").objectStore("projects").put({ ...project, id: "current", savedAt: Date.now() }));
}

export async function saveProjectSnapshot(project, label = "Snapshot") {
  const db = await openProjectDatabase();
  await request(db.transaction("snapshots", "readwrite").objectStore("snapshots").put({
    ...project, id: `${Date.now()}-${crypto.randomUUID?.() || Math.random()}`, label, savedAt: Date.now(),
  }));
}

export async function loadProjectSnapshots() {
  const db = await openProjectDatabase();
  const values = await request(db.transaction("snapshots", "readonly").objectStore("snapshots").getAll());
  return values.sort((left, right) => right.savedAt - left.savedAt);
}

export async function deleteProjectSnapshot(id) {
  const db = await openProjectDatabase();
  await request(db.transaction("snapshots", "readwrite").objectStore("snapshots").delete(id));
}

export async function savePreparedBackend(backend) {
  const db = await openProjectDatabase();
  await request(db.transaction("backends", "readwrite").objectStore("backends").put({ ...backend, savedAt: Date.now() }));
}

export async function loadPreparedBackends() {
  const db = await openProjectDatabase();
  return request(db.transaction("backends", "readonly").objectStore("backends").getAll());
}

export async function deletePreparedBackend(id) {
  const db = await openProjectDatabase();
  await request(db.transaction("backends", "readwrite").objectStore("backends").delete(id));
}

function openProjectDatabase() {
  if (!globalThis.indexedDB) return Promise.reject(new Error("IndexedDB is unavailable."));
  return new Promise((resolve, reject) => {
    const opening = indexedDB.open(DATABASE, VERSION);
    opening.onerror = () => reject(opening.error);
    opening.onupgradeneeded = () => {
      const db = opening.result;
      if (!db.objectStoreNames.contains("projects")) db.createObjectStore("projects", { keyPath: "id" });
      if (!db.objectStoreNames.contains("snapshots")) db.createObjectStore("snapshots", { keyPath: "id" });
      if (!db.objectStoreNames.contains("backends")) db.createObjectStore("backends", { keyPath: "id" });
    };
    opening.onsuccess = () => resolve(opening.result);
  });
}

function request(value) {
  return new Promise((resolve, reject) => {
    value.onsuccess = () => resolve(value.result);
    value.onerror = () => reject(value.error);
  });
}
