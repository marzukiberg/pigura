// Toast tunggal untuk seluruh aplikasi (dirender sekali oleh ui/Toast.svelte di App).
export const toast = $state({ msg: '', lift: 0 });

let timer;

/**
 * Tampilkan pesan singkat; hilang otomatis setelah `ms`.
 * `lift` menaikkan toast (px) bila ada elemen melayang lain, mis. dock aksi.
 */
export function showToast(msg, ms = 2200, lift = 0) {
  toast.msg = msg;
  toast.lift = lift;
  clearTimeout(timer);
  timer = setTimeout(() => (toast.msg = ''), ms);
}
