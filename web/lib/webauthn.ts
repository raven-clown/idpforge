// Bridges the browser's native WebAuthn API (which speaks ArrayBuffers)
// and go-webauthn's JSON wire format (which speaks base64url strings) --
// no external dependency, just the couple of encode/decode helpers this
// needs.

function base64urlToBuffer(base64url: string): ArrayBuffer {
  const padding = "=".repeat((4 - (base64url.length % 4)) % 4);
  const base64 = (base64url + padding).replace(/-/g, "+").replace(/_/g, "/");
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes.buffer;
}

function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i++) binary += String.fromCharCode(bytes[i]);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type JSONOptions = Record<string, any>;

// Runs the browser's "create a new credential" ceremony (registering a
// security key / platform authenticator) against options fetched from
// POST /api/v1/webauthn/register/begin, and returns a JSON body ready to
// POST to /api/v1/webauthn/register/finish.
export async function registerSecurityKey(options: JSONOptions): Promise<unknown> {
  // go-webauthn's JSON shape only differs from the browser's native type in
  // which fields are base64url strings vs. ArrayBuffers, so the safe cast
  // is "trust the server's shape, we're just swapping those fields" rather
  // than trying to convince the compiler this object structurally matches
  // PublicKeyCredentialCreationOptions field-by-field.
  const publicKey = {
    ...options,
    challenge: base64urlToBuffer(options.challenge),
    user: { ...options.user, id: base64urlToBuffer(options.user.id) },
    excludeCredentials: (options.excludeCredentials ?? []).map((c: JSONOptions) => ({
      ...c,
      id: base64urlToBuffer(c.id),
    })),
  } as unknown as PublicKeyCredentialCreationOptions;

  const credential = (await navigator.credentials.create({ publicKey })) as PublicKeyCredential | null;
  if (!credential) throw new Error("no credential returned");
  const response = credential.response as AuthenticatorAttestationResponse;

  return {
    id: credential.id,
    rawId: bufferToBase64url(credential.rawId),
    type: credential.type,
    response: {
      attestationObject: bufferToBase64url(response.attestationObject),
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
    },
    clientExtensionResults: credential.getClientExtensionResults(),
  };
}

export function webauthnSupported(): boolean {
  return typeof window !== "undefined" && !!window.PublicKeyCredential;
}
