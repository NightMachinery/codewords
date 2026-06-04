export function roomPath(roomId: string): string {
  return `/rooms/${encodeURIComponent(roomId)}`;
}

export function legacyRoomPath(roomId: string): string {
  return `/room/${encodeURIComponent(roomId)}`;
}

export function roomIdFromPath(pathname: string): string {
  const match = /^\/rooms?\/([^/]+)\/?$/.exec(pathname);
  return match ? decodeURIComponent(match[1]) : '';
}

export function canonicalHost(host: string): string {
  const url = new URL(`http://${host}`);
  url.hostname = url.hostname.replace(/\.+$/, '');
  return url.host;
}

export function canonicalOrigin(pageUrl: URL): string {
  return `${pageUrl.protocol}//${canonicalHost(pageUrl.host)}`;
}

export function websocketRoomUrl(
  pageUrl: URL,
  roomId: string,
  credential: { authToken?: string; migrateId?: string },
): string {
  const protocol = pageUrl.protocol === 'https:' ? 'wss:' : 'ws:';
  const url = new URL(`${protocol}//${canonicalHost(pageUrl.host)}/ws/rooms/${encodeURIComponent(roomId)}`);
  if (credential.migrateId) {
    url.searchParams.set('migrateId', credential.migrateId);
  } else if (credential.authToken) {
    url.searchParams.set('authToken', credential.authToken);
  }
  return url.toString();
}
