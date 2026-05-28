export type ConfirmationKind = 'reveal' | 'pass' | 'restart' | 'switchUnitySpymaster';
export type ConfirmationTone = 'reveal' | 'pass' | 'danger';

export interface ConfirmationCardPreview {
  label: string;
  imageUrl: string;
}

export interface RevealConfirmationCard {
  label: string;
  imageUrl?: string;
}

export interface ConfirmationRequest {
  kind: ConfirmationKind;
  title: string;
  message: string;
  confirmLabel: string;
  cancelLabel: string;
  tone: ConfirmationTone;
  cardPreview?: ConfirmationCardPreview;
}

export function buildRevealConfirmation(card: RevealConfirmationCard): ConfirmationRequest {
  return {
    kind: 'reveal',
    title: 'Reveal card?',
    message: `Reveal ${card.label} to the room.`,
    confirmLabel: 'Reveal',
    cancelLabel: 'Keep Hidden',
    tone: 'reveal',
    cardPreview: card.imageUrl ? { label: card.label, imageUrl: card.imageUrl } : undefined,
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

export function buildSwitchUnitySpymasterConfirmation(): ConfirmationRequest {
  return {
    kind: 'switchUnitySpymaster',
    title: 'Switch Unity spy?',
    message: 'Move the live Unity board to the next eligible player. If this spy has given a clue or revealed a card, this spends their turn.',
    confirmLabel: 'Switch Spy',
    cancelLabel: 'Stay Here',
    tone: 'pass',
  };
}
