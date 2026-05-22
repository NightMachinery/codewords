export type ConfirmationKind = 'reveal' | 'pass' | 'restart';
export type ConfirmationTone = 'reveal' | 'pass' | 'danger';

export interface ConfirmationRequest {
  kind: ConfirmationKind;
  title: string;
  message: string;
  confirmLabel: string;
  cancelLabel: string;
  tone: ConfirmationTone;
}

export function buildRevealConfirmation(cardLabel: string): ConfirmationRequest {
  return {
    kind: 'reveal',
    title: 'Reveal card?',
    message: `Reveal ${cardLabel} to the room.`,
    confirmLabel: 'Reveal',
    cancelLabel: 'Keep Hidden',
    tone: 'reveal',
  };
}

export function buildPassConfirmation(): ConfirmationRequest {
  return {
    kind: 'pass',
    title: 'Pass turn?',
    message: 'End guessing for this turn and hand play to the next team.',
    confirmLabel: 'Pass',
    cancelLabel: 'Keep Guessing',
    tone: 'pass',
  };
}

export function buildRestartConfirmation(): ConfirmationRequest {
  return {
    kind: 'restart',
    title: 'Restart match?',
    message: 'Return everyone to the lobby and generate a fresh board for the next start.',
    confirmLabel: 'Restart',
    cancelLabel: 'Stay Here',
    tone: 'danger',
  };
}
