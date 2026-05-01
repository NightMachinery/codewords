export type CopyResult =
  | { ok: true; method: 'clipboard' | 'execCommand' }
  | { ok: false; method: 'manual' };

interface ClipboardEnvironment {
  navigator?: {
    clipboard?: {
      writeText(text: string): Promise<void>;
    };
  };
  document?: {
    body?: {
      appendChild(node: unknown): void;
      removeChild(node: unknown): void;
    };
    createElement(tag: 'textarea'): {
      value: string;
      style: { position: string; left: string; top: string };
      focus(): void;
      select(): void;
    };
    execCommand(command: 'copy'): boolean;
  };
}

export async function copyText(text: string, env: ClipboardEnvironment = browserClipboardEnvironment()): Promise<CopyResult> {
  try {
    if (env.navigator?.clipboard?.writeText) {
      await env.navigator.clipboard.writeText(text);
      return { ok: true, method: 'clipboard' };
    }
  } catch {
    // Fall through to the HTTP-compatible execCommand path.
  }

  const doc = env.document;
  if (!doc?.body || !doc.createElement || !doc.execCommand) {
    return { ok: false, method: 'manual' };
  }

  const textarea = doc.createElement('textarea');
  textarea.value = text;
  textarea.style.position = 'fixed';
  textarea.style.left = '-9999px';
  textarea.style.top = '0';
  doc.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  try {
    return doc.execCommand('copy') ? { ok: true, method: 'execCommand' } : { ok: false, method: 'manual' };
  } catch {
    return { ok: false, method: 'manual' };
  } finally {
    doc.body.removeChild(textarea);
  }
}

function browserClipboardEnvironment(): ClipboardEnvironment {
  return {
    navigator: typeof navigator === 'undefined' ? undefined : navigator,
    document: typeof document === 'undefined' ? undefined : (document as unknown as ClipboardEnvironment['document']),
  };
}
