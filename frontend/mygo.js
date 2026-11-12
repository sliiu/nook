// Thin client for the Control service. Method names match what MyGo
// exposes as Service.Method; the runtime injects window.mygo.

function call(method, ...args) {
  return window.mygo.call(method, ...args);
}

export const Control = {
  getVolume() {
    return call("Control.GetVolume");
  },
  setVolume(level) {
    return call("Control.SetVolume", level);
  },
  setMute(muted) {
    return call("Control.SetMute", muted);
  },
  getBrightness() {
    return call("Control.GetBrightness");
  },
  setBrightness(level) {
    return call("Control.SetBrightness", level);
  },
  getState() {
    return call("Control.GetState");
  },
  listControls() {
    return call("Control.ListControls");
  },
  setControl(id, write) {
    return call("Control.SetControl", id, write || {});
  },
  noteDrag(dragging) {
    return call("Control.NoteDrag", dragging);
  },
  setFlyoutSize(w, h) {
    return call("Control.SetFlyoutSize", w, h);
  },
  getLang() {
    return call("Control.GetLang");
  },
  setLang(lang) {
    return call("Control.SetLang", lang);
  },
  noteIdle() {
    return call("Control.NoteIdle");
  },
  listScenes() {
    return call("Control.ListScenes");
  },
  getScene(id) {
    return call("Control.GetScene", id);
  },
  currentScene() {
    return call("Control.CurrentScene");
  },
  selectScene(id) {
    return call("Control.SelectScene", id);
  },
  saveScene(id, name, html) {
    return call("Control.SaveScene", id, name, html);
  },
  deleteScene(id) {
    return call("Control.DeleteScene", id);
  },
  restoreHome() {
    return call("Control.RestoreHome");
  },
  importHTML() {
    return call("Control.ImportHTML");
  },
  /**
   * Installs one catalog scene from a nook://scene/<id> link or a bare id.
   * Refreshes the scene when that catalog id is already installed.
   */
  installFromLink(link) {
    return call("Control.InstallFromLink", link);
  },
};

function on(name, listener) {
  return window.mygo.on(name, listener);
}

export const events = {
  stateChanged: { on: (listener) => on("state-changed", listener) },
  pillForm: { on: (listener) => on("pill-form", listener) },
  scenesChanged: { on: (listener) => on("scenes-changed", listener) },
  langChanged: { on: (listener) => on("lang-changed", listener) },
};
