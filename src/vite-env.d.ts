/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_ECOPASTE_TESTDATA_DIR?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
