export type Team = '' | 'blue' | 'red';

export interface LobbyPlayer {
  id: string;
  displayName: string;
  team: Team;
  spymaster: boolean;
  representative: boolean;
  mod: boolean;
}

export interface ViewerContext {
  userId: string;
  isHost: boolean;
  isMod?: boolean;
}

export function playerBuckets(players: LobbyPlayer[]): {
  blue: LobbyPlayer[];
  red: LobbyPlayer[];
  unassigned: LobbyPlayer[];
} {
  return {
    blue: players.filter((player) => player.team === 'blue'),
    red: players.filter((player) => player.team === 'red'),
    unassigned: players.filter((player) => player.team !== 'blue' && player.team !== 'red'),
  };
}

export function canManageLobby(viewer: ViewerContext | null | undefined): boolean {
  return Boolean(viewer?.isHost || viewer?.isMod);
}

export function startReadiness(players: LobbyPlayer[]): { ready: boolean; reason: string } {
  if (players.length === 0) {
    return { ready: false, reason: 'Invite at least one player first.' };
  }
  if (players.some((player) => player.team !== 'blue' && player.team !== 'red')) {
    return { ready: false, reason: 'Assign every player to a team first.' };
  }
  const blueSpy = players.some((player) => player.team === 'blue' && player.spymaster);
  const redSpy = players.some((player) => player.team === 'red' && player.spymaster);
  if (!blueSpy || !redSpy) {
    return { ready: false, reason: 'Each team needs a spymaster.' };
  }
  return { ready: true, reason: '' };
}
