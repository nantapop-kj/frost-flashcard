import { callService as call, onEvent } from './lib/api.js';
import { $, confirmClick, el, plural, relativeDays, showError, showMessage } from './lib/helpers.js';

const prefs = {
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
    } catch {}
  },
};

const isComposing = (e) => e.isComposing || e.keyCode === 229;
const Views = ['review', 'add', 'words', 'decks'];
let currentView = 'review';

const showView = (name) => {
  currentView = name;
  for (const v of Views) $(`view-${v}`).hidden = v !== name;
  document.querySelectorAll('nav button').forEach((b) =>
    b.setAttribute('aria-selected', String(b.dataset.view === name))
  );
  refreshView();
};

const refreshView = () => {
  if (currentView === 'review') startSession();
  if (currentView === 'add') {
    refreshCount();
    document.querySelector('#add-form [name=front]').focus();
  }
  if (currentView === 'words') loadWords();
  if (currentView === 'decks') renderDeckList();
};

document.querySelectorAll('nav button').forEach((b) =>
  b.addEventListener('click', () => showView(b.dataset.view))
);

let decks = { active: 0, decks: [] };
const deckName = (d) => d.name || 'My deck';

const refreshDecks = async () => {
  try {
    decks = await call('ListDecks');
  } catch (err) {
    showError('ListDecks', err);
    return;
  }
  const select = $('deck-select');
  select.replaceChildren(
    ...decks.decks.map((d) => {
      const opt = el('option', '', `${deckName(d)} (${d.due} due / ${d.count})`);
      opt.value = d.id;
      return opt;
    })
  );
  select.value = decks.active;
  if (currentView === 'decks') renderDeckList();
};

$('deck-select').addEventListener('change', async (e) => {
  try {
    await call('SetActiveDeck', Number(e.target.value));
  } catch (err) {
    showError('SetActiveDeck', err);
  }
  await refreshDecks();
  refreshView();
});

const renderDeckList = () => {
  $('deck-list').replaceChildren(
    ...decks.decks.map((d) => {
      const li = el('li', d.id === decks.active ? 'active' : '');
      const name = el('button', 'name link', deckName(d));
      name.dir = 'auto';
      name.title = 'Click to rename';
      name.addEventListener('click', () => startRename(name, d));

      const del = el('button', 'danger', 'Delete');
      confirmClick(del, async () => {
        try {
          await call('DeleteDeck', d.id);
          showMessage(`Deleted deck "${deckName(d)}".`);
        } catch (err) {
          showError('DeleteDeck', err);
        }
        await refreshDecks();
      });

      li.append(name, el('span', 'meta', plural(d.count, 'card')), del);
      return li;
    })
  );
};

const startRename = (button, deck) => {
  const input = el('input');
  input.value = deck.name;
  input.dir = 'auto';
  button.replaceWith(input);
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
    if (e.key === 'Escape') {
      e.stopPropagation();
      finish(false);
    }
  });
  input.addEventListener('blur', () => finish(true));
};

$('new-deck-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const form = e.currentTarget;
  const name = form.elements.name.value;
  try {
    await call('CreateDeck', name);
    form.reset();
    showMessage(`Created "${name.trim()}". Add some words to it.`);
    await refreshDecks();
    showView('add');
  } catch (err) {
    showError('CreateDeck', err);
  }
});

$('export').addEventListener('click', async () => {
  try {
    const report = await call('ExportCSV', 'Export deck as CSV');
    if (report) showMessage(`Exported ${plural(report.count, 'card')}.`);
  } catch (err) {
    showError('ExportCSV', err);
  }
});

const session = { cards: [], pos: 0, flipped: false, busy: false };
let filter = prefs.get('filter', 'due');
$('filter').value = filter;

$('filter').addEventListener('change', (e) => {
  filter = e.target.value;
  prefs.set('filter', filter);
  startSession();
});

const startSession = async () => {
  try {
    session.cards = await call('StartSession', filter);
  } catch (err) {
    session.cards = [];
    showError('StartSession', err);
  }
  session.pos = 0;
  renderCard();
  refreshStats();
};

const refreshStats = async () => {
  try {
    const s = await call('Stats');
    $('stats').textContent =
      `${s.due_today} due · ${s.new_cards} new · ${s.learning} learning · ${s.known} known · ${s.total} total`;
  } catch (err) {
    showError('Stats', err);
  }
};

const renderCard = () => {
  const card = session.cards[session.pos];
  session.flipped = false;

  $('card').hidden = !card;
  $('hint').hidden = !card;
  $('done').hidden = !!card;
  $('grades').hidden = true;
  $('card').classList.remove('flipped');
  $('remaining').textContent = card ? `${session.cards.length - session.pos} left` : '';
  if (!card) {
    const emptyFromStart = session.cards.length === 0;
    $('done').textContent =
      filter === 'due' || !emptyFromStart
        ? 'No cards due right now. Come back later!'
        : 'No cards match this filter.';
    return;
  }

  $('card-front').textContent = card.front;
  $('card-reading').textContent = card.reading;
  $('card-back').textContent = card.back;
};

const flipCard = () => {
  if (session.flipped || !session.cards[session.pos]) return;
  session.flipped = true;
  $('card').classList.add('flipped');
  $('hint').hidden = true;
  $('grades').hidden = false;
};

const gradeCard = async (grade) => {
  const card = session.cards[session.pos];
  if (!card || !session.flipped || session.busy) return;
  session.busy = true;
  try {
    const days = await call('GradeCard', card.id, grade);
    showMessage(`You'll see this card again ${relativeDays(days)}.`);
    session.pos++;
    renderCard();
    refreshStats();
    refreshDecks();
  } catch (err) {
    showError('GradeCard', err);
    if (err?.cause?.code === 'not_found') {
      session.pos++;
      renderCard();
    }
  } finally {
    session.busy = false;
  }
};

$('card').addEventListener('click', flipCard);
document.querySelectorAll('#grades button').forEach((b) =>
  b.addEventListener('click', () => gradeCard(b.dataset.grade))
);

const GradeKeys = { Digit1: 'again', Digit2: 'hard', Digit3: 'good', Digit4: 'easy' };

document.addEventListener('keydown', (e) => {
  if (currentView !== 'review' || ['INPUT', 'SELECT'].includes(e.target.tagName)) return;
  if (e.code === 'Space' || e.code === 'Enter') {
    e.preventDefault();
    flipCard();
  } else if (GradeKeys[e.code]) {
    gradeCard(GradeKeys[e.code]);
  }
});


const refreshCount = async () => {
  try {
    const s = await call('Stats');
    $('card-count').textContent = `${plural(s.total, 'card')} in this deck`;
  } catch (err) {
    showError('Stats', err);
  }
};

$('add-form').addEventListener('keydown', (e) => {
  if (e.key === 'Enter' && isComposing(e)) e.preventDefault();
});

$('add-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const form = e.currentTarget;
  const { front, back, reading } = Object.fromEntries(new FormData(form));
  try {
    await call('AddCard', front, back, reading);
    form.reset();
    form.elements.front.focus();
    showMessage(`Added "${front.trim()}".`);
    refreshCount();
    refreshDecks();
  } catch (err) {
    showError('AddCard', err);
  }
});

$('import').addEventListener('click', async () => {
  try {
    const report = await call('ImportFile', 'Import words from Excel or CSV');
    if (!report) return;
    const parts = [`Imported ${plural(report.added, 'word')}`];
    if (report.duplicates) parts.push(`${report.duplicates} already in deck`);
    if (report.empty) parts.push(`${plural(report.empty, 'row')} missing word or meaning`);
    showMessage(parts.join(' · '), false, 5000);
    refreshCount();
    refreshDecks();
  } catch (err) {
    showError('ImportFile', err);
  }
});

let allCards = [];
let editingId = null;

const loadWords = async () => {
  try {
    allCards = await call('ListCards');
  } catch (err) {
    showError('ListCards', err);
    allCards = [];
  }
  renderWords();
};

const norm = (s) => s.normalize('NFC').toLocaleLowerCase();

const daysUntil = (iso) => {
  const [y, m, d] = iso.split('-').map(Number);
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  return Math.round((new Date(y, m - 1, d) - today) / 86_400_000);
};

const renderWords = () => {
  const q = norm($('search').value.trim());
  const matches = allCards.filter(
    (c) => !q || [c.front, c.back, c.reading].some((s) => norm(s).includes(q))
  );

  const rows = matches.map((card) => (card.id === editingId ? editRow(card) : wordRow(card)));
  if (!matches.length) {
    rows.push(el('li', 'meta', q ? 'No words match your search.' : 'No words in this deck yet.'));
  }
  $('card-list').replaceChildren(...rows);
};

const wordRow = (card) => {
  const li = el('li', 'clickable');
  li.title = 'Click to edit';
  li.addEventListener('click', () => {
    editingId = card.id;
    renderWords();
  });

  const words = el('div', 'words');
  const front = el('div', '', card.front);
  front.dir = 'auto';
  if (card.reading) front.append(el('span', 'meta', ` ${card.reading}`));
  const back = el('div', 'meta', card.back);
  back.dir = 'auto';
  words.append(front, back);

  const days = daysUntil(card.due);
  const due = el('span', `meta status-${card.status}`, days <= 0 ? 'due' : relativeDays(days));
  due.title = card.status;

  const del = el('button', 'danger', 'Delete');
  confirmClick(del, async () => {
    try {
      await call('DeleteCard', card.id);
      showMessage(`Deleted "${card.front}".`);
    } catch (err) {
      showError('DeleteCard', err);
    }
    loadWords();
    refreshDecks();
  });

  li.append(words, due, del);
  return li;
};

const editRow = (card) => {
  const li = el('li', 'editing');
  const form = el('form');
  const field = (name, value, placeholder) => {
    const input = el('input');
    Object.assign(input, { name, value, placeholder, dir: 'auto', autocomplete: 'off' });
    return input;
  };
  const cancel = el('button', '', 'Cancel');
  cancel.type = 'button';
  const save = el('button', '', 'Save');
  save.type = 'submit';
  const actions = el('div', 'row');
  actions.append(cancel, save);
  form.append(
    field('front', card.front, 'Word'),
    field('reading', card.reading, 'Reading (optional)'),
    field('back', card.back, 'Meaning'),
    actions
  );
  li.append(form);

  const close = () => {
    editingId = null;
    renderWords();
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
    const { front, back, reading } = Object.fromEntries(new FormData(form));
    try {
      await call('UpdateCard', card.id, front, back, reading);
      showMessage(`Saved "${front.trim()}".`);
      editingId = null;
      loadWords();
    } catch (err) {
      showError('UpdateCard', err);
    }
  });

  requestAnimationFrame(() => form.elements.front.focus());
  return li;
};

$('search').addEventListener('input', renderWords);

onEvent('navigate', async (view) => {
  await refreshDecks();
  showView(view === 'add' ? 'add' : 'review');
});

await refreshDecks();
showView('review');
