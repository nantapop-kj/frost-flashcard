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

let messageTimer;
export const showMessage = (text, isError = false, ms = 3000) => {
  const el = $('message');
  el.textContent = text;
  el.classList.toggle('error', isError);
  clearTimeout(messageTimer);
  messageTimer = setTimeout(() => (el.textContent = ''), ms);
};

export const showError = (action, err) => {
  console.error(`${action} failed:`, err?.cause ?? err);
  showMessage(ErrorText[errorCode(err)] ?? ErrorText.unknown, true, 5000);
};

export const relativeDays = (days) =>
  new Intl.RelativeTimeFormat('en', { numeric: 'auto' }).format(days, 'day');

export const plural = (n, word) => `${n} ${word}${n === 1 ? '' : 's'}`;

export const el = (tag, className, text) => {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
};

export const confirmClick = (button, onConfirm) => {
  const label = button.textContent;
  let timer;
  const disarm = () => {
    clearTimeout(timer);
    delete button.dataset.armed;
    button.textContent = label;
  };
  button.addEventListener('click', async (e) => {
    e.stopPropagation();
    if (!button.dataset.armed) {
      button.dataset.armed = 'true';
      button.textContent = 'Sure?';
      timer = setTimeout(disarm, 3000);
      return;
    }
    disarm();
    await onConfirm();
  });
};
