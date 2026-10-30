/**
 * idea.config — the live config the web app builds against.
 *
 * Defaults to the ScopeLock worked example so the skeleton builds and runs as a
 * working POC out of the box. To create a NEW poc, swap the re-export below to
 * point at your own config in ../configs (or replace this file's contents with
 * a filled-in copy of ../idea.config.example.ts). Swapping the config = a new POC.
 */
export type { IdeaConfig, OutputFormat, ValidationSignal } from "./idea.config.types";
export { ideaConfig } from "./configs/scopelock.config";
