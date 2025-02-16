/**
 * Digest represents a set of attributes that briefly describe the underlying
 * transport implementation.
 *
 * @field protocol - a string description of the protocol being used by the
 * freighter.
 * @encoder - a string description of the encoder being used by the freighter.
 */
export interface Digest {
  protocol: string;
  encoder: string;
}

// Payload represents a piece of data that can be sent over the freighter.
export type Payload = unknown;
