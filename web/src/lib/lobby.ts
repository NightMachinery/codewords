export type Team = '' | 'blue' | 'red' | 'observers';

export interface LobbyPlayer {
  id: string;
  displayName: string;
  team: Team;
  spymaster: boolean;
  representative: boolean;
  mod: boolean;
  previousTeam?: Team;
  previousSpymaster?: boolean;
  previousRepresentative?: boolean;
}

export interface ViewerContext {
  userId: string;
  isHost: boolean;
  isMod?: boolean;
}

export function playerBuckets(players: LobbyPlayer[]): {
  blue: LobbyPlayer[];
  red: LobbyPlayer[];
  observers: LobbyPlayer[];
  unassigned: LobbyPlayer[];
} {
  return {
    blue: players.filter((player) => player.team === 'blue'),
    red: players.filter((player) => player.team === 'red'),
    observers: players.filter((player) => player.team === 'observers'),
    unassigned: players.filter((player) => player.team === ''),
  };
}

export type VisiblePlayerBucket = { tone: 'blue' | 'red' | 'observers' | 'unassigned'; members: LobbyPlayer[] };

export function visiblePlayerBuckets(players: LobbyPlayer[]): VisiblePlayerBucket[] {
  const buckets = playerBuckets(players);
  return [
    { tone: 'blue' as const, members: buckets.blue },
    { tone: 'red' as const, members: buckets.red },
    { tone: 'observers' as const, members: buckets.observers },
    { tone: 'unassigned' as const, members: buckets.unassigned },
  ].filter((bucket) => bucket.tone === 'blue' || bucket.tone === 'red' || bucket.members.length > 0);
}

export function canManageLobby(viewer: ViewerContext | null | undefined): boolean {
  return Boolean(viewer?.isHost || viewer?.isMod);
}

export function startReadiness(players: LobbyPlayer[]): { ready: boolean; reason: string } {
  if (players.length === 0) {
    return { ready: false, reason: 'Invite at least one player first.' };
  }
  if (players.some((player) => player.team === '')) {
    return { ready: false, reason: 'Assign every player to a team or observer mode first.' };
  }
  const blueSpy = players.some((player) => player.team === 'blue' && player.spymaster);
  const redSpy = players.some((player) => player.team === 'red' && player.spymaster);
  if (!blueSpy || !redSpy) {
    return { ready: false, reason: 'Each team needs a spymaster.' };
  }
  const blueGuesser = players.some((player) => player.team === 'blue' && !player.spymaster);
  const redGuesser = players.some((player) => player.team === 'red' && !player.spymaster);
  if (!blueGuesser || !redGuesser) {
    return { ready: false, reason: 'Each team needs a non-spymaster guesser.' };
  }
  return { ready: true, reason: '' };
}
