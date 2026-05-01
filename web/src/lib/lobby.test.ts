import { describe, expect, it } from 'vitest';

import { canManageLobby, playerBuckets, startReadiness, type LobbyPlayer } from './lobby';

const players: LobbyPlayer[] = [
  { id: 'host', displayName: 'Host', team: 'blue', spymaster: true, representative: false },
  { id: 'guest', displayName: 'Guest', team: 'red', spymaster: true, representative: false },
  { id: 'floater', displayName: 'Floater', team: '', spymaster: false, representative: false },
];

describe('lobby helpers', () => {
  it('groups players by team assignment', () => {
    expect(playerBuckets(players)).toEqual({
      blue: [players[0]],
      red: [players[1]],
      unassigned: [players[2]],
    });
  });

  it('allows only the host to manage settings and roles', () => {
    expect(canManageLobby({ userId: 'host', isHost: true })).toBe(true);
    expect(canManageLobby({ userId: 'guest', isHost: false })).toBe(false);
  });

  it('explains what prevents the host from starting', () => {
    expect(startReadiness(players)).toEqual({ ready: false, reason: 'Assign every player to a team first.' });
    expect(startReadiness(players.slice(0, 2))).toEqual({ ready: true, reason: '' });
    expect(startReadiness([{ ...players[0], spymaster: false }, players[1]])).toEqual({
      ready: false,
      reason: 'Each team needs a spymaster.',
    });
  });
});
