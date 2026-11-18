import { errorCode } from './api.js';

export const $ = (id) => document.getElementById(id);

const ErrorText = {
  empty_fields: 'Please fill in both "Word" and "Meaning".',
  not_found: 'This card is no longer in the deck. It may have been deleted.',
  save_failed: 'Could not save to disk. Your last change was undone.',
  bad_input: 'The app sent invalid data. This is a bug in the app.',
  unknown: 'Something went wrong. Please try again.',
};

let messageTimer;
export const showMessage = (text, isError = false) => {
  const el = $('message');
  el.textContent = text;
  el.classList.toggle('error', isError);
  clearTimeout(messageTimer);
  messageTimer = setTimeout(() => (el.textContent = ''), 3000);
};

export const showError = (action, err) => {
  console.error(`${action} failed:`, err?.cause ?? err);
  showMessage(ErrorText[errorCode(err)] ?? ErrorText.unknown, true);
};

export const relativeDays = (days) =>
  new Intl.RelativeTimeFormat('en', { numeric: 'auto' }).format(days, 'day');
