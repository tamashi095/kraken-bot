/**
 * Kraken API Type Definitions
 * 
 * This file contains all TypeScript interfaces and types used throughout the application.
 */

/**
 * Request data structure that must include a nonce
 */
export interface KrakenRequestData {
  nonce: number | string;
  [key: string]: unknown;
}

/**
 * Input data can be either a JSON string or an object with nonce
 */
export type KrakenDataInput = string | KrakenRequestData;

/**
 * Balance response from Kraken API
 */
export type BalanceResponse = Record<string, string>;

/**
 * Order parameters for AddOrder endpoint
 */
export interface AddOrderParams {
  /** Order direction (buy/sell) */
  type: "buy" | "sell";
  /** The execution model of the order */
  ordertype:
    | "market"
    | "limit"
    | "iceberg"
    | "stop-loss"
    | "take-profit"
    | "stop-loss-limit"
    | "take-profit-limit"
    | "trailing-stop"
    | "trailing-stop-limit"
    | "settle-position";
  /** Asset pair id or altname */
  pair: string;
  /** Order quantity in terms of the base asset */
  volume: string;
  /** Limit price for limit orders */
  price?: string;
  /** Secondary price for stop orders */
  price2?: string;
  /** Price signal used to trigger stop orders */
  trigger?: "index" | "last";
  /** Amount of leverage desired */
  leverage?: string;
  /** If true, order will only reduce a currently open position */
  reduce_only?: boolean;
  /** Self Trade Prevention mode */
  stptype?: "cancel-newest" | "cancel-oldest" | "cancel-both";
  /** Comma delimited list of order flags */
  oflags?: string;
  /** Time-in-force of the order */
  timeinforce?: "GTC" | "IOC" | "GTD";
  /** Scheduled start time */
  starttm?: string;
  /** Expiry time */
  expiretm?: string;
  /** User reference id */
  userref?: number;
  /** Client order identifier */
  cl_ord_id?: string;
  /** Validate order only (don't execute) */
  validate?: boolean;
  /** Deadline for order execution */
  deadline?: string;
  [key: string]: unknown;
}

/**
 * Response from AddOrder endpoint
 */
export interface AddOrderResponse {
  descr: {
    order: string;
    close?: string;
  };
  txid?: string[];
}

/**
 * Withdrawal method information
 */
export interface WithdrawMethod {
  asset: string;
  method: string;
  network?: string;
  minimum?: string;
  [key: string]: unknown;
}

/**
 * Parameters for withdrawal request
 */
export interface WithdrawParams {
  /** Asset being withdrawn */
  asset: string;
  /** Withdrawal key name, as set up on your account */
  key: string;
  /** Amount to be withdrawn */
  amount: string;
  /** Crypto address that can be used to confirm address matches key */
  address?: string;
  /** Asset class of the asset being withdrawn */
  aclass?: "currency" | "tokenized_asset";
  /** If the processed withdrawal fee is higher than max_fee, withdrawal will fail */
  max_fee?: string;
  /** Optional parameter for viewing xstocks data */
  rebase_multiplier?: "rebased" | "base";
  [key: string]: unknown;
}

/**
 * Response from Withdraw endpoint
 */
export interface WithdrawResponse {
  refid: string;
}

/**
 * Configuration options for KrakenClient
 */
export interface KrakenClientConfig {
  apiKey: string;
  secret: string;
  baseUrl?: string;
}

/**
 * Options for making a request to the Kraken API
 */
export interface RequestOptions {
  method?: "GET" | "POST";
  body?: Record<string, unknown>;
  otp?: string;
}

