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
  noteDrag(dragging) {
    return call("Control.NoteDrag", dragging);
  },
  noteIdle() {
    return call("Control.NoteIdle");
  },
};

function on(name, listener) {
  return window.mygo.on(name, listener);
}

export const events = {
  stateChanged: { on: (listener) => on("state-changed", listener) },
  pillForm: { on: (listener) => on("pill-form", listener) },
};
