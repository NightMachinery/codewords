export type Team = '' | 'blue' | 'red' | 'unity' | 'monality' | 'observers';

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
  playerId?: string;
  isHost: boolean;
  isMod?: boolean;
}

export type PlayerPanelPhase = 'lobby' | 'active' | 'game_over';

export function playerBuckets(players: LobbyPlayer[]): {
  blue: LobbyPlayer[];
  red: LobbyPlayer[];
  unity: LobbyPlayer[];
  monality: LobbyPlayer[];
  observers: LobbyPlayer[];
  unassigned: LobbyPlayer[];
} {
  return {
    blue: players.filter((player) => player.team === 'blue'),
    red: players.filter((player) => player.team === 'red'),
    unity: players.filter((player) => player.team === 'unity'),
    monality: players.filter((player) => player.team === 'monality'),
    observers: players.filter((player) => player.team === 'observers'),
    unassigned: players.filter((player) => player.team === ''),
  };
}

export type VisiblePlayerBucket = { tone: 'blue' | 'red' | 'unity' | 'monality' | 'observers' | 'unassigned'; members: LobbyPlayer[] };

export function visiblePlayerBuckets(players: LobbyPlayer[]): VisiblePlayerBucket[] {
  const buckets = playerBuckets(players);
  return [
    { tone: 'blue' as const, members: buckets.blue },
    { tone: 'red' as const, members: buckets.red },
    { tone: 'unity' as const, members: buckets.unity },
    { tone: 'monality' as const, members: buckets.monality },
    { tone: 'observers' as const, members: buckets.observers },
    { tone: 'unassigned' as const, members: buckets.unassigned },
  ].filter((bucket) => bucket.tone === 'blue' || bucket.tone === 'red' || bucket.members.length > 0);
}

export function isViewerPlayer(player: Pick<LobbyPlayer, 'id'>, viewer: Pick<ViewerContext, 'userId' | 'playerId'> | null | undefined): boolean {
  return Boolean(viewer && player.id === (viewer.playerId || viewer.userId));
}

export function canShowTeamAssignmentButton(input: {
  phase: PlayerPanelPhase;
  mode?: 'polarity' | 'unity' | 'monality';
  hostControls: boolean;
  player: LobbyPlayer;
  viewer: Pick<ViewerContext, 'userId' | 'playerId'> | null | undefined;
  team: Team;
}): boolean {
  if (input.phase === 'game_over') return false;
  const mode = input.mode ?? 'polarity';
  if (mode === 'unity') {
    return input.hostControls && (input.team === 'unity' || input.team === 'observers') && input.player.team !== input.team;
  }
  if (mode === 'monality') {
    return input.hostControls && (input.team === 'monality' || input.team === 'observers') && input.player.team !== input.team;
  }
  if (input.team === 'unity' || input.team === 'monality') return false;
  if (input.hostControls) return true;
  if (!isViewerPlayer(input.player, input.viewer)) return false;
  if (input.phase === 'lobby') return true;
  return input.team === 'observers' && input.player.team !== 'observers';
}

export function canShowRejoinTeamButton(input: {
  phase: PlayerPanelPhase;
  mode?: 'polarity' | 'unity' | 'monality';
  hostControls: boolean;
  player: LobbyPlayer;
  viewer: Pick<ViewerContext, 'userId' | 'playerId'> | null | undefined;
}): boolean {
  const mode = input.mode ?? 'polarity';
  return input.phase !== 'game_over'
    && input.player.team === 'observers'
    && (mode === 'unity' ? (input.player.previousTeam === 'blue' || input.player.previousTeam === 'red' || input.player.previousTeam === 'unity') : mode === 'monality' ? input.player.previousTeam === 'monality' : (input.player.previousTeam === 'blue' || input.player.previousTeam === 'red'))
    && (input.hostControls || isViewerPlayer(input.player, input.viewer));
}

export function canShowRoleControls(input: { phase: PlayerPanelPhase; hostControls: boolean; player: LobbyPlayer }): boolean {
  return input.phase !== 'game_over' && input.hostControls && (input.player.team === 'blue' || input.player.team === 'red' || input.player.team === 'unity');
}

export function canShowModControl(input: { phase: PlayerPanelPhase; hostControls: boolean; player: LobbyPlayer; roomHostId: string }): boolean {
  return input.phase !== 'game_over' && input.hostControls && input.player.id !== input.roomHostId;
}

export function canShowMigrateDeviceButton(input: { phase: PlayerPanelPhase; hostControls: boolean; player: LobbyPlayer }): boolean {
  return input.phase !== 'game_over' && input.hostControls && Boolean(input.player.id);
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
