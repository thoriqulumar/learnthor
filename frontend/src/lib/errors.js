// Central friendly-error mapper. Every page should call friendlyError(err)
// instead of hand-writing .catch() strings, so users see consistent,
// human-readable, bilingual messages instead of raw backend codes.

// Message table: backend error code → { en, id }.
const BY_CODE = {
  INVALID_CREDENTIALS: {
    en: 'Email or password is incorrect.',
    id: 'Email atau kata sandi salah.',
  },
  EMAIL_OR_USERNAME_TAKEN: {
    en: 'That email or username is already registered.',
    id: 'Email atau nama pengguna itu sudah terdaftar.',
  },
  UNAUTHENTICATED: {
    en: 'Please log in to continue.',
    id: 'Silakan masuk untuk melanjutkan.',
  },
  INVALID_TOKEN: {
    en: 'Your session has expired — please log in again.',
    id: 'Sesi kamu sudah berakhir — silakan masuk lagi.',
  },
  UPGRADE_REQUIRED: {
    en: 'This section is part of Premium. Unlock it to continue learning.',
    id: 'Bagian ini termasuk konten Premium. Buka untuk melanjutkan belajar.',
  },
  ALREADY_PREMIUM: {
    en: 'You already have Premium access.',
    id: 'Kamu sudah memiliki akses Premium.',
  },
  MODULE_LOCKED: {
    en: 'Master the previous module first to unlock this one.',
    id: 'Selesaikan modul sebelumnya dulu untuk membuka modul ini.',
  },
  FORBIDDEN: {
    en: "You don't have access to this content yet.",
    id: 'Kamu belum punya akses ke konten ini.',
  },
  NOT_FOUND: {
    en: "We couldn't find what you're looking for.",
    id: 'Kami tidak menemukan yang kamu cari.',
  },
  INVALID_PAYLOAD: {
    en: 'Something was off with that request — please try again.',
    id: 'Ada yang salah dengan permintaan itu — coba lagi ya.',
  },
  INVALID_STATE: {
    en: "That action isn't available right now.",
    id: 'Aksi itu belum tersedia saat ini.',
  },
  INVALID_RATING: {
    en: 'Please choose a rating from 1 to 4.',
    id: 'Silakan pilih penilaian dari 1 sampai 4.',
  },
  INVALID_QUIZ_TYPE: {
    en: "That quiz type isn't available.",
    id: 'Jenis kuis itu tidak tersedia.',
  },
  INVALID_MODULE_ID: {
    en: 'That link looks broken — please go back and try again.',
    id: 'Tautan itu sepertinya rusak — kembali lalu coba lagi.',
  },
  INVALID_ID: {
    en: 'That link looks broken — please go back and try again.',
    id: 'Tautan itu sepertinya rusak — kembali lalu coba lagi.',
  },
  INVALID_PREFERENCE: {
    en: "That setting isn't valid.",
    id: 'Pengaturan itu tidak valid.',
  },
  INVALID_SIGNATURE: {
    en: 'Payment verification failed. If you were charged, contact support.',
    id: 'Verifikasi pembayaran gagal. Jika kamu sudah membayar, hubungi dukungan.',
  },
  INTERNAL_ERROR: {
    en: 'Something went wrong on our side. Please try again.',
    id: 'Terjadi kesalahan di sisi kami. Silakan coba lagi.',
  },
};

// Status-based fallbacks when no specific code is known.
const BY_STATUS = {
  400: { en: 'Something was off with that request — please try again.', id: 'Ada yang salah dengan permintaan itu — coba lagi ya.' },
  401: { en: 'Please log in to continue.', id: 'Silakan masuk untuk melanjutkan.' },
  403: { en: "You don't have access to this content yet.", id: 'Kamu belum punya akses ke konten ini.' },
  404: { en: "We couldn't find what you're looking for.", id: 'Kami tidak menemukan yang kamu cari.' },
  409: { en: 'That action conflicts with the current state.', id: 'Aksi itu bertentangan dengan keadaan saat ini.' },
  429: { en: 'Too many attempts — wait a moment and try again.', id: 'Terlalu banyak percobaan — tunggu sebentar lalu coba lagi.' },
};

const NETWORK = {
  en: "Can't reach the server — check your connection and try again.",
  id: 'Tidak bisa menghubungi server — periksa koneksi kamu lalu coba lagi.',
};

const UNKNOWN = {
  en: 'Something went wrong. Please try again.',
  id: 'Terjadi kesalahan. Silakan coba lagi.',
};

// friendlyErrorParts returns both language variants for an error, choosing
// the most specific message available: error code → status → network → generic.
export function friendlyErrorParts(err) {
  // Network-level failure (no response at all).
  if (!err?.response) {
    return NETWORK;
  }

  const code = err.response.data?.code;
  if (code && BY_CODE[code]) {
    return BY_CODE[code];
  }

  const status = err.response.status;
  if (status >= 500) {
    return BY_CODE.INTERNAL_ERROR;
  }
  if (BY_STATUS[status]) {
    return BY_STATUS[status];
  }
  return UNKNOWN;
}

// friendlyError returns a single message string for an error in the user's
// preferred language. lang follows the app's preferred_gloss values:
// 'id' → Indonesian; 'en' and 'both' → English (UI chrome is English-first).
export function friendlyError(err, lang = 'en') {
  const parts = friendlyErrorParts(err);
  return lang === 'id' ? parts.id : parts.en;
}
