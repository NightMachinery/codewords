import { describe, expect, it } from 'vitest';

import { canManageLobby, canShowModControl, canShowRejoinTeamButton, canShowRoleControls, canShowTeamAssignmentButton, playerBuckets, visiblePlayerBuckets, startReadiness, type LobbyPlayer } from './lobby';

const players: LobbyPlayer[] = [
  { id: 'host', displayName: 'Host', team: 'blue', spymaster: true, representative: false, mod: true },
  { id: 'guest', displayName: 'Guest', team: 'red', spymaster: true, representative: false, mod: false },
  { id: 'floater', displayName: 'Floater', team: '', spymaster: false, representative: false, mod: false },
];

describe('lobby helpers', () => {
  it('groups players by team assignment', () => {
    expect(playerBuckets(players)).toEqual({
      blue: [players[0]],
      red: [players[1]],
      observers: [],
      unassigned: [players[2]],
    });
  });

  it('hides empty observer and unassigned panels from the rendered buckets', () => {
    expect(visiblePlayerBuckets(players).map((bucket) => bucket.tone)).toEqual(['blue', 'red', 'unassigned']);
    expect(visiblePlayerBuckets(players.slice(0, 2)).map((bucket) => bucket.tone)).toEqual(['blue', 'red']);
    expect(visiblePlayerBuckets([...players, { id: 'obs', displayName: 'Observer', team: 'observers', spymaster: false, representative: false, mod: false }]).map((bucket) => bucket.tone)).toEqual(['blue', 'red', 'observers', 'unassigned']);
  });

  it('allows hosts and promoted mods to manage settings and roles', () => {
    expect(canManageLobby({ userId: 'host', isHost: true, isMod: true })).toBe(true);
    expect(canManageLobby({ userId: 'guest', isHost: false, isMod: true })).toBe(true);
    expect(canManageLobby({ userId: 'guest', isHost: false, isMod: false })).toBe(false);
  });

  it('shows mid-game player controls to moderators and only self observer/rejoin controls to regular players', () => {
    const viewer = { userId: 'guest', isHost: false, isMod: false };
    const modViewer = { userId: 'host', isHost: false, isMod: true };
    const guest = players[1];
    const observer = { ...players[1], team: 'observers' as const, previousTeam: 'red' as const };

    expect(canShowTeamAssignmentButton({ phase: 'active', hostControls: true, player: guest, viewer: modViewer, team: 'blue' })).toBe(true);
    expect(canShowRoleControls({ phase: 'active', hostControls: true, player: guest })).toBe(true);
    expect(canShowModControl({ phase: 'active', hostControls: true, player: guest, roomHostId: 'host' })).toBe(true);

    expect(canShowTeamAssignmentButton({ phase: 'active', hostControls: false, player: guest, viewer, team: 'observers' })).toBe(true);
    expect(canShowTeamAssignmentButton({ phase: 'active', hostControls: false, player: guest, viewer, team: 'blue' })).toBe(false);
    expect(canShowTeamAssignmentButton({ phase: 'active', hostControls: false, player: players[0], viewer, team: 'observers' })).toBe(false);
    expect(canShowRejoinTeamButton({ phase: 'active', hostControls: false, player: observer, viewer })).toBe(true);
  });

  it('explains what prevents the host from starting', () => {
    expect(startReadiness(players)).toEqual({ ready: false, reason: 'Assign every player to a team or observer mode first.' });
    const startable = [players[0], { ...players[0], id: 'blue-guess', spymaster: false }, players[1], { ...players[1], id: 'red-guess', spymaster: false }];
    expect(startReadiness(startable)).toEqual({ ready: true, reason: '' });
    expect(startReadiness(players.slice(0, 2))).toEqual({ ready: false, reason: 'Each team needs a non-spymaster guesser.' });
    expect(startReadiness([{ ...players[0], spymaster: false }, players[1]])).toEqual({
      ready: false,
      reason: 'Each team needs a spymaster.',
    });
  });
});
