import { callService as call, callSystem, errorCode, hideWindow, onEvent } from './lib/api.js';
import {
  $,
  confirmClick,
  confirmDialog,
  daysUntil,
  el,
  isComposing,
  plural,
  prefs,
  relativeDays,
  requireFilled,
  setInvalid,
  showError,
  toast,
} from './lib/helpers.js';

const ICON_X =
  '<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M6 6l12 12M18 6 6 18"/></svg>';

const FilterNames = {
  all: 'All cards',
  due: 'Due today',
  new: 'New',
  learning: 'Learning',
  known: 'Known',
};

const deckName = (deck) => deck.name || 'My deck';

const darkQuery = window.matchMedia('(prefers-color-scheme: dark)');

const applyAppearance = (value) => {
  const mode = value === 'system' ? (darkQuery.matches ? 'dark' : 'light') : value;
  document.documentElement.dataset.mode = mode;
  document.querySelectorAll('#appearance button').forEach((b) =>
    b.classList.toggle('active', b.dataset.value === value)
  );
};

document.querySelectorAll('#appearance button').forEach((btn) =>
  btn.addEventListener('click', () => {
    prefs.set('appearance', btn.dataset.value);
    applyAppearance(btn.dataset.value);
  })
);

darkQuery.addEventListener('change', () => {
  if (prefs.get('appearance', 'system') === 'system') applyAppearance('system');
});

const applyReadingMode = (mode) => {
  $('card').dataset.reading = mode;
  document.querySelectorAll('#reading-mode button').forEach((b) =>
    b.classList.toggle('active', b.dataset.value === mode)
  );
};

document.querySelectorAll('#reading-mode button').forEach((btn) =>
  btn.addEventListener('click', () => {
    prefs.set('reading', btn.dataset.value);
    applyReadingMode(btn.dataset.value);
  })
);

const showView = (name) => {
  document.querySelectorAll('.tab-link').forEach((b) =>
    b.classList.toggle('active', b.dataset.view === name)
  );
  document.querySelectorAll('.view').forEach((v) =>
    v.classList.toggle('active', v.id === `view-${name}`)
  );
  $('progress').classList.toggle('hidden', name !== 'review');
  closeDeckMenu();
  document.querySelectorAll('input.invalid').forEach((input) => setInvalid(input, false));

  if (name === 'list') loadList();
  if (name === 'settings') loadAppInfo();
  if (name === 'add') $('front').focus();
};

const currentView = () => document.querySelector('.view.active').id.replace('view-', '');

document.querySelectorAll('.tab-link').forEach((btn) =>
  btn.addEventListener('click', () => showView(btn.dataset.view))
);

let decks = { active: 0, decks: [] };

const refreshDecks = async () => {
  try {
    decks = await call('ListDecks');
  } catch (err) {
    showError('ListDecks', err);
    return;
  }
  renderDecks();
};

const renderDecks = () => {
  const active = decks.decks.find((d) => d.id === decks.active);
  if (!active) return;

  $('deck-name').textContent = deckName(active);
  $('deck-count').textContent = active.due
    ? `${active.due} due · ${plural(active.count, 'card')}`
    : plural(active.count, 'card');

  $('deck-options').replaceChildren(
    ...decks.decks.map((deck) => {
      const btn = el('button');
      btn.setAttribute('role', 'menuitem');
      const name = el('span', 'name', deckName(deck));
      name.dir = 'auto';
      btn.append(
        el('span', 'check', deck.id === decks.active ? '✓' : ''),
        name,
        el('span', 'count', plural(deck.count, 'card'))
      );
      btn.addEventListener('click', () => switchDeck(deck.id));
      const li = el('li');
      li.append(btn);
      return li;
    })
  );

  renderDeckManager();
};

const switchDeck = async (id) => {
  closeDeckMenu();
  if (id === decks.active) return;
  try {
    await call('SetActiveDeck', id);
    await refreshDecks();
    await startSession();
    if (currentView() === 'list') loadList();
  } catch (err) {
    showError('SetActiveDeck', err);
  }
};

const openDeckMenu = () => {
  $('deck-menu').classList.remove('hidden');
  $('deck-button').setAttribute('aria-expanded', 'true');
};

const closeDeckMenu = () => {
  $('deck-menu').classList.add('hidden');
  $('deck-button').setAttribute('aria-expanded', 'false');
};

$('deck-button').addEventListener('click', (e) => {
  e.stopPropagation();
  $('deck-menu').classList.contains('hidden') ? openDeckMenu() : closeDeckMenu();
});
$('deck-menu').addEventListener('click', (e) => e.stopPropagation());
document.addEventListener('click', closeDeckMenu);

$('new-deck-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const name = $('new-deck-name').value.trim();
  try {
    await call('CreateDeck', name);
    $('new-deck-name').value = '';
    closeDeckMenu();
    toast(`Created "${name}".`);
    await refreshDecks();
    await startSession();
    showView('add');
  } catch (err) {
    showError('CreateDeck', err);
  }
});

const renderDeckManager = () => {
  $('deck-manage').replaceChildren(
    ...decks.decks.map((deck) => {
      const li = el('li', deck.id === decks.active ? 'active' : '');

      const name = el('button', 'name', deckName(deck));
      name.dir = 'auto';
      name.title = 'Rename';
      name.addEventListener('click', () => startRename(name, deck));

      const del = el('button', 'del');
      del.innerHTML = ICON_X;
      del.title = 'Delete deck';
      del.setAttribute('aria-label', del.title);
      confirmClick(
        del,
        async () => {
          try {
            await call('DeleteDeck', deck.id);
            toast(`Deleted "${deckName(deck)}".`);
            await refreshDecks();
            await startSession();
          } catch (err) {
            showError('DeleteDeck', err);
          }
        },
        'Click again to delete this deck and all its cards.'
      );

      li.append(name, el('span', 'count', plural(deck.count, 'card')), del);
      return li;
    })
  );
};

const startRename = (nameButton, deck) => {
  const input = el('input', 'rename');
  input.value = deck.name;
  input.dir = 'auto';
  nameButton.replaceWith(input);
  input.focus();
  input.select();

  let done = false;
  const finish = async (save) => {
    if (done) return;
    done = true;
    const name = input.value.trim();
    if (save && name !== deck.name) {
      try {
        await call('RenameDeck', deck.id, name);
      } catch (err) {
        showError('RenameDeck', err);
      }
    }
    await refreshDecks();
  };

  input.addEventListener('keydown', (e) => {
    if (isComposing(e)) return;
    if (e.key === 'Enter') finish(true);
    if (e.key === 'Escape') finish(false);
  });
  input.addEventListener('blur', () => finish(true));
};

let session = { queue: [], pos: 0 };
let filter = prefs.get('filter', 'due');

const startSession = async () => {
  try {
    session = { queue: await call('StartSession', filter), pos: 0 };
  } catch (err) {
    session = { queue: [], pos: 0 };
    showError('StartSession', err);
  }
  renderSession();
};

const currentCard = () => session.queue[session.pos];

const setText = (node, text, longAt, shortAt = 0) => {
  const length = [...text].length;
  node.textContent = text;
  node.classList.toggle('long', length > longAt);
  node.classList.toggle('short', length <= shortAt);
};

const renderSession = () => {
  const card = currentCard();
  const total = session.queue.length;
  const hasCard = card !== undefined;

  $('card').classList.toggle('hidden', !hasCard);
  $('hint').classList.toggle('hidden', !hasCard);
  $('done').classList.toggle('hidden', hasCard);
  $('grades').classList.add('hidden');
  $('card').classList.remove('flipped');

  const badge = $('filter-badge');
  badge.classList.toggle('hidden', filter === 'due');
  badge.textContent = FilterNames[filter];
  badge.dataset.status = filter;

  $('progress-bar').style.width = total ? `${(session.pos / total) * 100}%` : '0';

  if (!hasCard) {
    $('remaining').textContent = '';
    const emptyByFilter = total === 0 && filter !== 'due';
    $('done-title').textContent = emptyByFilter ? 'No cards match this filter' : 'All done for now';
    $('done-sub').textContent = emptyByFilter
      ? 'Choose another filter in Settings.'
      : 'Come back later, or add some new words.';
    return;
  }

  $('remaining').textContent = `${total - session.pos} left`;
  setText($('card-front'), card.front, 16, 3);
  $('card-reading').textContent = card.reading;
  setText($('card-back'), card.back, 60);
};

const flipCard = () => {
  if (!currentCard()) return;
  $('card').classList.add('flipped');
  $('hint').classList.add('hidden');
  $('grades').classList.remove('hidden');
};

let grading = false;

const gradeCard = async (grade) => {
  const card = currentCard();
  if (!card || grading) return;
  grading = true;
  try {
    const days = await call('GradeCard', card.id, grade);
    toast(`You'll see this card again ${relativeDays(days)}.`);
  } catch (err) {
    showError('GradeCard', err);
    if (errorCode(err) !== 'not_found') {
      grading = false;
      return;
    }
  }
  session.pos += 1;
  renderSession();
  refreshDecks();
  grading = false;
};

$('card').addEventListener('click', flipCard);
document.querySelectorAll('#grades button').forEach((btn) =>
  btn.addEventListener('click', () => gradeCard(btn.dataset.grade))
);

const renderFilterOptions = () => {
  document.querySelectorAll('#filter button').forEach((b) => {
    const on = b.dataset.value === filter;
    b.classList.toggle('active', on);
    b.setAttribute('aria-checked', String(on));
  });
};

document.querySelectorAll('#filter button').forEach((btn) =>
  btn.addEventListener('click', () => {
    filter = btn.dataset.value;
    prefs.set('filter', filter);
    renderFilterOptions();
    startSession();
  })
);

const GradeKeys = { Digit1: 'again', Digit2: 'hard', Digit3: 'good', Digit4: 'easy' };

document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') closeDeckMenu();
  if (currentView() !== 'review' || ['INPUT', 'SELECT'].includes(e.target.tagName)) return;
  if (e.code === 'Space' || e.code === 'Enter') {
    e.preventDefault();
    flipCard();
    return;
  }
  if (GradeKeys[e.code] && $('card').classList.contains('flipped')) {
    gradeCard(GradeKeys[e.code]);
  }
});

for (const form of [$('add-form'), $('new-deck-form')]) {
  form.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && isComposing(e)) e.preventDefault();
  });
}

$('add-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const required = [$('front'), $('back')];
  if (!requireFilled(required)) return;
  const front = $('front').value;
  try {
    await call('AddCard', front, $('back').value, $('reading').value);
    toast(`Added "${front.trim()}".`);
    $('add-form').reset();
    $('front').focus();
    refreshDecks();
  } catch (err) {
    if (errorCode(err) === 'empty_fields') requireFilled(required);
    else showError('AddCard', err);
  }
});

$('import').addEventListener('click', async () => {
  try {
    const report = await call('ImportFile', 'Import words from Excel or CSV');
    if (!report) return;
    const parts = [`Imported ${plural(report.added, 'word')}`];
    if (report.duplicates) parts.push(`${report.duplicates} already in deck`);
    if (report.empty) parts.push(`${plural(report.empty, 'row')} missing word or meaning`);
    toast(parts.join(' · '), 4000);
    refreshDecks();
  } catch (err) {
    showError('ImportFile', err);
  }
});

let allCards = [];
let listFilter = 'all';
let editingId = null;

const loadList = async () => {
  try {
    allCards = await call('ListCards');
  } catch (err) {
    allCards = [];
    showError('ListCards', err);
  }
  renderChips();
  renderList();
};

const renderChips = () => {
  $('chips').replaceChildren(
    ...['all', 'new', 'learning', 'known'].map((status) => {
      const n = status === 'all' ? allCards.length : allCards.filter((c) => c.status === status).length;
      const btn = el('button', status === listFilter ? 'active' : '', status === 'all' ? 'All' : FilterNames[status]);
      if (status !== 'all') btn.dataset.status = status;
      btn.append(el('span', 'n', String(n)));
      btn.addEventListener('click', () => {
        listFilter = status;
        renderChips();
        renderList();
      });
      return btn;
    })
  );
};

const norm = (s) => s.normalize('NFC').toLocaleLowerCase();

const renderList = () => {
  const q = norm($('search').value.trim());
  const matches = allCards.filter(
    (c) =>
      (listFilter === 'all' || c.status === listFilter) &&
      (!q || [c.front, c.back, c.reading].some((s) => norm(s).includes(q)))
  );

  const rows = matches.map((card) => (card.id === editingId ? editRow(card) : wordRow(card)));
  if (!matches.length) {
    rows.push(
      el('li', 'empty', q || listFilter !== 'all' ? 'No words match.' : 'No words in this deck yet.')
    );
  }
  $('card-list').replaceChildren(...rows);
};

const wordRow = (card) => {
  const li = el('li', 'clickable');
  li.title = 'Click to edit';
  li.addEventListener('click', () => {
    editingId = card.id;
    renderList();
  });

  const dot = el('span', 'dot');
  dot.dataset.status = card.status;
  dot.title = FilterNames[card.status];

  const words = el('div', 'words');
  const front = el('div', '', card.front);
  front.dir = 'auto';
  if (card.reading) front.append(el('span', 'rd', card.reading));
  const back = el('div', 'meaning', card.back);
  back.dir = 'auto';
  words.append(front, back);

  const days = daysUntil(card.due);
  const due = el('span', 'due', days <= 0 ? 'due' : relativeDays(days));

  const del = el('button', 'del');
  del.innerHTML = ICON_X;
  del.title = 'Delete word';
  del.setAttribute('aria-label', del.title);
  confirmClick(
    del,
    async () => {
      try {
        await call('DeleteCard', card.id);
        toast(`Deleted "${card.front}".`);
        loadList();
        refreshDecks();
      } catch (err) {
        showError('DeleteCard', err);
      }
    },
    'Click again to delete this word.'
  );

  li.append(dot, words, due, del);
  return li;
};

const editRow = (card) => {
  const li = el('li', 'editing');
  const form = el('form', 'edit-form');

  const field = (value, placeholder) => {
    const input = el('input', 'glass');
    Object.assign(input, { value, placeholder, dir: 'auto', autocomplete: 'off' });
    return input;
  };
  const front = field(card.front, 'Word');
  const reading = field(card.reading, 'Reading');
  const back = field(card.back, 'Meaning');

  const cancel = el('button', 'mini ghost', 'Cancel');
  cancel.type = 'button';
  const save = el('button', 'mini', 'Save');
  save.type = 'submit';

  const row = el('div', 'row');
  row.append(front, reading);
  const actions = el('div', 'actions');
  actions.append(cancel, save);
  form.append(row, back, actions);
  li.append(form);

  const close = () => {
    editingId = null;
    renderList();
  };

  cancel.addEventListener('click', close);
  form.addEventListener('keydown', (e) => {
    if (isComposing(e)) {
      if (e.key === 'Enter') e.preventDefault();
      return;
    }
    if (e.key === 'Escape') {
      e.stopPropagation();
      close();
    }
  });
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!requireFilled([front, back])) return;
    const values = [front.value.trim(), back.value.trim(), reading.value.trim()];
    try {
      await call('UpdateCard', card.id, ...values);
      const inSession = session.queue.find((c) => c.id === card.id);
      if (inSession) [inSession.front, inSession.back, inSession.reading] = values;
      toast(`Saved "${values[0]}".`);
      editingId = null;
      loadList();
    } catch (err) {
      showError('UpdateCard', err);
    }
  });

  requestAnimationFrame(() => {
    front.focus();
    front.select();
  });
  return li;
};

$('search').addEventListener('input', renderList);

let appInfo = { version: '', platform: '', canInstall: false };

const loadAppInfo = async () => {
  try {
    appInfo = await callSystem('AppInfo');
  } catch (err) {
    showError('AppInfo', err);
    return;
  }
  $('install-app').classList.toggle('hidden', !appInfo.canInstall);
  $('version').textContent = `Frost Flashcard ${appInfo.version}`;
  const extra = {
    darwin: 'The app is then removed from your Mac.',
    windows: 'The app is then removed from your PC.',
  }[appInfo.platform];
  $('uninstall-help').textContent = ['Uninstall deletes all your decks and words.', extra]
    .filter(Boolean)
    .join(' ');
  if (appInfo.platform === 'windows') {
    $('quit-help').textContent =
      'To quit, right-click the tray icon (under ^ on the taskbar) and choose Quit.';
  }
};

$('export').addEventListener('click', async () => {
  try {
    const report = await call('ExportCSV', 'Export deck as CSV');
    if (report) toast(`Exported ${plural(report.count, 'card')}.`);
  } catch (err) {
    showError('ExportCSV', err);
  }
});

$('install-app').addEventListener('click', async () => {
  try {
    await callSystem('InstallApp');
  } catch (err) {
    showError('InstallApp', err);
  }
});

$('uninstall').addEventListener('click', async () => {
  const sure = await confirmDialog({
    title: 'Uninstall Frost Flashcard?',
    text: 'All your decks and words will be deleted. This cannot be undone. Export a deck to CSV first if you want to keep it.',
    ok: 'Uninstall',
  });
  if (!sure) return;
  try {
    await callSystem('Uninstall');
  } catch (err) {
    showError('Uninstall', err);
  }
});

const refreshAll = async () => {
  await refreshDecks();
  await startSession();
  if (currentView() === 'list') loadList();
};

onEvent('popup-shown', refreshAll);

onEvent('navigate', async (view) => {
  await refreshAll();
  showView(view === 'add' ? 'add' : 'review');
});

if (/Windows/.test(navigator.userAgent)) document.documentElement.dataset.platform = 'windows';
$('close-window').addEventListener('click', () => hideWindow());
applyAppearance(prefs.get('appearance', 'system'));
applyReadingMode(prefs.get('reading', 'answer'));
renderFilterOptions();
await refreshAll();
