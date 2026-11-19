import { callService as call, onEvent } from './lib/api.js';
import { $, relativeDays, showError, showMessage } from './lib/helpers.js';

const showView = (name) => {
  $('view-review').hidden = name !== 'review';
  $('view-add').hidden = name !== 'add';
  document.querySelectorAll('nav button').forEach((b) =>
    b.setAttribute('aria-selected', String(b.dataset.view === name))
  );
  if (name === 'review') startSession();
  else {
    refreshCount();
    document.querySelector('#add-form [name=front]').focus();
  }
};

document.querySelectorAll('nav button').forEach((b) =>
  b.addEventListener('click', () => showView(b.dataset.view))
);

const session = { cards: [], pos: 0, flipped: false, busy: false };

const startSession = async () => {
  try {
    session.cards = await call('StartSession', 'due');
  } catch (err) {
    session.cards = [];
    showError('StartSession', err);
  }
  session.pos = 0;
  renderCard();
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
  if (!card) return;

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
  } catch (err) {
    showError('GradeCard', err);
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
  if ($('view-review').hidden || e.target.tagName === 'INPUT') return;
  if (e.code === 'Space' || e.code === 'Enter') {
    e.preventDefault();
    flipCard();
  } else if (GradeKeys[e.code]) {
    gradeCard(GradeKeys[e.code]);
  }
});

const refreshCount = async () => {
  try {
    const cards = await call('ListCards');
    $('card-count').textContent = `${cards.length} ${cards.length === 1 ? 'card' : 'cards'} in this deck`;
  } catch (err) {
    showError('ListCards', err);
  }
};

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
  } catch (err) {
    showError('AddCard', err);
  }
});

onEvent('navigate', (view) => showView(view === 'add' ? 'add' : 'review'));

showView('review');
