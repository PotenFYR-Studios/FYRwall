// Shared copy-to-clipboard button used by code blocks and the hero
// install command. Shows "Copied!" for ~1.4s (SPEC 5.8).
import { useState } from "react";

export function CopyButton({ text }: { text: string }) {
  const [ok, setOk] = useState(false);
  return (
    <button
      type="button"
      className={`copy-btn static${ok ? " ok" : ""}`}
      style={{ opacity: 1 }}
      onClick={() => {
        navigator.clipboard?.writeText(text).then(
          () => {
            setOk(true);
            setTimeout(() => setOk(false), 1400);
          },
          () => undefined
        );
      }}
    >
      {ok ? "Copied!" : "Copy"}
    </button>
  );
}
