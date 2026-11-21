import { errorCode } from './api.js';

export const $ = (id) => document.getElementById(id);

const ErrorText = {
  empty_fields: 'Please fill in both "Word" and "Meaning".',
  empty_name: 'Please type a deck name.',
  not_found: 'This item is no longer there. It may have been deleted.',
  last_deck: 'You cannot delete your only deck. Create another deck first.',
  save_failed: 'Could not save to disk. Your last change was undone.',
  unsupported_file: 'Only .xlsx, .xlsm and .csv files can be imported.',
  not_utf8: 'This CSV is not UTF-8. In Excel, save it as "CSV UTF-8" and try again.',
  no_sheet: 'This Excel file has no sheets.',
  read_failed: 'Could not read that file.',
  write_failed: 'Could not write the file. Check the folder and try again.',
  bad_input: 'The app sent invalid data. This is a bug in the app.',
  unknown: 'Something went wrong. Please try again.',
};

export const errorMessage = (err) => ErrorText[errorCode(err)] ?? ErrorText.unknown;

let toastTimer;
export const toast = (text, ms = 2200) => {
  const el = $('toast');
  el.textContent = text;
  el.classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => el.classList.remove('show'), ms);
};

export const showError = (action, err) => {
  console.error(`${action} failed:`, err?.cause ?? err);
  toast(errorMessage(err), 5000);
};

export const relativeDays = (days) =>
  new Intl.RelativeTimeFormat('en', { numeric: 'auto' }).format(days, 'day');

export const daysUntil = (iso) => {
  const [y, m, d] = iso.split('-').map(Number);
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  return Math.round((new Date(y, m - 1, d) - today) / 86_400_000);
};

export const plural = (n, word) => `${n} ${word}${n === 1 ? '' : 's'}`;

export const el = (tag, className, text) => {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
};

export const requireFilled = (inputs) => {
  const empty = inputs.filter((input) => !input.value.trim());
  for (const input of inputs) setInvalid(input, empty.includes(input));
  if (empty.length) {
    empty[0].focus();
    toast(errorMessage({ cause: { code: 'empty_fields' } }), 3000);
  }
  return empty.length === 0;
};

export const setInvalid = (input, invalid) => {
  input.classList.toggle('invalid', invalid);
  if (invalid) {
    input.setAttribute('aria-invalid', 'true');
    input.addEventListener('input', () => setInvalid(input, false), { once: true });
  } else {
    input.removeAttribute('aria-invalid');
  }
};

export const isComposing = (e) => e.isComposing || e.keyCode === 229;

export const confirmClick = (button, onConfirm, message = 'Click again to confirm.') => {
  let armed = false;
  let timer;
  button.addEventListener('click', async (e) => {
    e.stopPropagation();
    if (!armed) {
      armed = true;
      button.classList.add('confirm');
      toast(message, 3000);
      timer = setTimeout(() => {
        armed = false;
        button.classList.remove('confirm');
      }, 3000);
      return;
    }
    clearTimeout(timer);
    armed = false;
    button.classList.remove('confirm');
    await onConfirm();
  });
};

export const confirmDialog = ({ title, text, ok }) =>
  new Promise((resolve) => {
    const modal = $('confirm');
    $('confirm-title').textContent = title;
    $('confirm-text').textContent = text;
    $('confirm-ok').textContent = ok;
    modal.classList.remove('hidden');
    $('confirm-cancel').focus();

    const close = (answer) => {
      modal.classList.add('hidden');
      $('confirm-ok').removeEventListener('click', onOk);
      $('confirm-cancel').removeEventListener('click', onCancel);
      modal.removeEventListener('click', onBackdrop);
      document.removeEventListener('keydown', onKey, true);
      resolve(answer);
    };
    const onOk = () => close(true);
    const onCancel = () => close(false);
    const onBackdrop = (e) => {
      if (e.target === modal) close(false);
    };
    const onKey = (e) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        e.preventDefault();
        close(false);
      }
    };

    $('confirm-ok').addEventListener('click', onOk);
    $('confirm-cancel').addEventListener('click', onCancel);
    modal.addEventListener('click', onBackdrop);
    document.addEventListener('keydown', onKey, true);
  });

export const prefs = {
  get(key, fallback) {
    try {
      return localStorage.getItem(`frost-flashcard.${key}`) ?? fallback;
    } catch {
      return fallback;
    }
  },
  set(key, value) {
    try {
      localStorage.setItem(`frost-flashcard.${key}`, value);
    } catch { }
  },
};
