package render

// quizAttemptJS is the client-side logic for the quiz attempt page. It reads
// question data from a JSON script block, renders one question at a time, times
// each answer via performance.now(), submits via fetch, and shows inline
// feedback. No JS framework — vanilla DOM manipulation matching the site.
const quizAttemptJS = `
(function() {
  var data = JSON.parse(document.getElementById('attempt-data').textContent);
  var questions = data.questions;
  var attemptId = data.attemptId;
  var answeredIds = {};
  var results = {};
  var responses = {};
  data.answeredIds.forEach(function(id) {
    answeredIds[id] = true;
    var r = data.answeredResults[String(id)];
    results[id] = r === true;
  });

  var currentIdx = 0;
  for (var i = 0; i < questions.length; i++) {
    if (!answeredIds[questions[i].id]) { currentIdx = i; break; }
    currentIdx = Math.min(i + 1, questions.length - 1);
  }
  if (currentIdx >= questions.length) currentIdx = questions.length - 1;

  var renderStartTime = 0;
  var submitted = false;
  var correctCount = 0;
  var answeredCount = 0;
  for (var id in answeredIds) { answeredCount++; if (results[id]) correctCount++; }
  var letters = 'ABCDEFG';

  function renderDots() {
    var c = document.getElementById('attempt-dots');
    if (!c) return;
    c.innerHTML = '';
    for (var i = 0; i < questions.length; i++) {
      var d = document.createElement('span');
      d.className = 'w-2 h-2 rounded-full cursor-pointer hover:ring-2 hover:ring-slate-300 ';
      if (answeredIds[questions[i].id]) {
        d.className += results[questions[i].id] ? 'bg-emerald-600' : 'bg-red-600';
      } else if (i === currentIdx) {
        d.className += 'bg-blue-700 ring-2 ring-blue-700/40';
      } else {
        d.className += 'bg-slate-200';
      }
      d.onclick = (function(n) { return function() { if (n !== currentIdx) { currentIdx = n; renderQuestion(); } }; })(i);
      c.appendChild(d);
    }
  }

  function getSelectedIndex() {
    var btns = document.querySelectorAll('#question-area .option-btn');
    for (var i = 0; i < btns.length; i++) {
      if (btns[i].classList.contains('border-blue-700')) return i;
    }
    return -1;
  }

  function answeredChoiceHTML(q, sel, correctIdx) {
    var h = '<div class="space-y-2">';
    for (var i = 0; i < q.options.length; i++) {
      if (i === correctIdx) {
        h += '<div class="flex items-center gap-3 p-3 rounded-lg border-2 border-emerald-600 bg-emerald-100 text-sm text-slate-800 font-medium">';
        h += '<span class="w-6 h-6 rounded-full bg-emerald-600 text-white flex items-center justify-center text-xs shrink-0">' + letters[i] + '</span>';
        h += '<span class="font-medium text-slate-900">Correct:</span> ' + q.options[i];
        h += '<svg class="ml-auto shrink-0 text-emerald-600" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"/></svg>';
        h += '</div>';
      } else if (i === sel) {
        h += '<div class="flex items-center gap-3 p-3 rounded-lg border-2 border-red-600 bg-red-100 text-sm text-slate-800 font-medium">';
        h += '<span class="w-6 h-6 rounded-full bg-red-600 text-white flex items-center justify-center text-xs shrink-0">' + letters[i] + '</span>';
        h += '<span class="font-medium text-slate-900">You answered:</span> ' + q.options[i];
        h += '<svg class="ml-auto shrink-0 text-red-600" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>';
        h += '</div>';
      } else {
        h += '<div class="flex items-center gap-3 p-3 rounded-lg border border-slate-200 text-sm text-slate-400">';
        h += '<span class="w-6 h-6 rounded-full border border-slate-300 flex items-center justify-center text-xs text-slate-400 shrink-0">' + letters[i] + '</span>';
        h += q.options[i];
        h += '</div>';
      }
    }
    h += '</div>';
    return h;
  }

  function renderQuestion() {
    submitted = false;
    var q = questions[currentIdx];
    renderStartTime = performance.now();
    var card = document.getElementById('question-area');
    var isAnswered = !!answeredIds[q.id];
    var resp = responses[q.id];
    var html = '';

    if (q.mode === 'choice') {
      if (isAnswered && resp) {
        html = '<h3 class="text-lg font-medium text-slate-800 leading-relaxed mb-5">' + q.title + '</h3>';
        html += answeredChoiceHTML(q, resp.selected, resp.correctIndex);
      } else {
        html = '<h3 class="text-lg font-medium text-slate-800 leading-relaxed mb-5">' + q.title + '</h3>';
        html += '<div class="space-y-2" id="options">';
        for (var i = 0; i < q.options.length; i++) {
          html += '<button data-idx="' + i + '" class="option-btn w-full text-left flex items-center gap-3 p-3 rounded-lg border border-slate-200 hover:bg-slate-50 hover:border-slate-300 transition-colors cursor-pointer text-sm text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-700">';
          html += '<span class="w-6 h-6 rounded-full border border-slate-300 flex items-center justify-center text-xs text-slate-500 shrink-0">' + letters[i] + '</span>';
          html += q.options[i] + '</button>';
        }
        html += '</div>';
      }
    } else if (q.mode === 'recall') {
      if (isAnswered && resp) {
        html = '<div class="p-6 rounded-lg border border-slate-200 bg-white"><p class="text-lg font-medium text-slate-800 leading-relaxed mb-3">' + q.title + '</p>';
        html += '<div class="p-4 rounded-lg border border-dashed border-slate-300 bg-slate-50 text-sm text-slate-700 leading-relaxed mb-4">' + q.reveal + '</div>';
        html += '<div class="text-xs ' + (resp.isCorrect ? 'text-emerald-600' : 'text-red-600') + '">' + (resp.isCorrect ? 'You marked this as known.' : 'You marked this for review.') + '</div></div>';
      } else {
        html = '<div id="flashcard" tabindex="0" role="button" aria-label="Flip card" class="relative cursor-pointer select-none focus:outline-none focus:ring-2 focus:ring-blue-700 rounded-lg" style="perspective:1000px" onclick="var i=document.getElementById(\'flashcard-inner\');if(!document.getElementById(\'flashcard\').dataset.locked){i.style.transform=i.style.transform?\'\':\'rotateY(180deg)\';i.classList.toggle(\'flipped\');if(i.classList.contains(\'flipped\')&&!document.getElementById(\'flashcard\').dataset.gradeShown){window._showGradeButtons()}}">';
        html += '<div id="flashcard-inner" class="relative transition-transform duration-500" style="transform-style:preserve-3d;display:grid">';
        html += '<div id="card-front" class="flex items-center justify-center p-6 rounded-lg border border-slate-200 bg-white text-center" style="backface-visibility:hidden;grid-area:1/1;min-height:200px">';
        html += '<div><p class="text-lg font-medium text-slate-800 leading-relaxed">' + q.title + '</p>';
        html += '<p class="text-xs text-slate-400 italic mt-4">Tap to reveal answer</p></div>';
        html += '</div>';
        html += '<div id="card-back" aria-hidden="true" class="flex items-center justify-center p-6 rounded-lg border border-dashed border-slate-300 bg-slate-50 text-center overflow-y-auto" style="backface-visibility:hidden;transform:rotateY(180deg);grid-area:1/1;min-height:200px;max-height:400px">';
        html += '<p class="text-base text-slate-700 leading-relaxed">' + q.reveal + '</p>';
        html += '</div>';
        html += '</div></div>';
      }
    }
    card.innerHTML = html;

    var actions = document.getElementById('attempt-actions');
    if (actions) actions.innerHTML = '';
    var scoreLine = document.getElementById('score-line');
    if (scoreLine) scoreLine.textContent = '';

    if (isAnswered) {
      showNextButton();
    } else if (q.mode === 'choice') {
      var btns = card.querySelectorAll('.option-btn');
      btns.forEach(function(btn) {
        btn.addEventListener('click', function() {
          btns.forEach(function(b) { b.classList.remove('border-blue-700', 'border-2'); });
          btn.classList.add('border-blue-700', 'border-2');
          var sb = document.getElementById('submit-btn');
          sb.disabled = false;
          sb.className = 'bg-blue-700 text-white text-sm font-medium py-2 px-4 rounded-lg hover:bg-blue-700/90 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-700';
        });
      });
      showChoiceActions(q);
    } else if (q.mode === 'recall') {
      document.getElementById('flashcard').addEventListener('keydown', function(e) {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          if (!this.dataset.locked) {
            var inner = document.getElementById('flashcard-inner');
            inner.style.transform = inner.style.transform ? '' : 'rotateY(180deg)';
            inner.classList.toggle('flipped');
            if (inner.classList.contains('flipped') && !this.dataset.gradeShown) {
              showGradeButtons();
            }
          }
        }
      });
      renderStartTime = performance.now();
      showRecallActions();
    }
    updateNav();
    renderDots();
  }

  function showChoiceActions(q) {
    var actions = document.getElementById('attempt-actions');
    if (!actions) return;
    var btn = document.createElement('button');
    btn.id = 'submit-btn';
    btn.disabled = true;
    btn.className = 'bg-blue-700 text-white text-sm font-medium py-2 px-4 rounded-lg cursor-not-allowed transition-colors disabled:opacity-40 disabled:cursor-not-allowed';
    btn.textContent = 'Check';
    btn.addEventListener('click', function() {
      if (submitted) return;
      var sel = getSelectedIndex();
      if (sel < 0) return;
      submitChoice(q, sel);
    });
    actions.appendChild(btn);
  }

  function showRecallActions() {
  }

  function showGradeButtons() {
    var fc = document.getElementById('flashcard');
    if (fc.dataset.gradeShown) return;
    fc.dataset.gradeShown = '1';
    renderStartTime = performance.now();
    var q = questions[currentIdx];
    var actions = document.getElementById('attempt-actions');
    if (!actions) return;
    actions.innerHTML = '';
    var gradeDiv = document.createElement('div');
    gradeDiv.className = 'flex gap-3';
    gradeDiv.innerHTML = '<button id="got-it-btn" class="bg-emerald-600 text-white text-sm font-medium py-2 px-4 rounded-lg hover:bg-emerald-600/90 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-emerald-600">Got it</button>'
      + '<button id="not-yet-btn" class="bg-red-600 text-white text-sm font-medium py-2 px-4 rounded-lg hover:bg-red-600/90 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-red-600">Not yet</button>';
    actions.appendChild(gradeDiv);
    document.getElementById('got-it-btn').focus();
    document.getElementById('got-it-btn').addEventListener('click', function() { if (submitted) return; submitRecall(q, true); });
    document.getElementById('not-yet-btn').addEventListener('click', function() { if (submitted) return; submitRecall(q, false); });
  }
  window._showGradeButtons = showGradeButtons;

  function disableButtons() {
    var submitBtn = document.getElementById('submit-btn');
    if (submitBtn) submitBtn.disabled = true;
    var gotIt = document.getElementById('got-it-btn');
    if (gotIt) gotIt.disabled = true;
    var notYet = document.getElementById('not-yet-btn');
    if (notYet) notYet.disabled = true;
  }

  function showError(msg) {
    var actions = document.getElementById('attempt-actions');
    if (!actions) { actions = document.getElementById('question-area'); }
    var err = document.createElement('div');
    err.className = 'text-xs text-red-600';
    err.textContent = msg || 'Something went wrong. Please try again.';
    actions.appendChild(err);
  }

  function updateNav() {
    var prevBtn = document.getElementById('attempt-prev');
    if (prevBtn) prevBtn.style.visibility = (currentIdx === 0) ? 'hidden' : 'visible';
  }

  function submitChoice(q, sel) {
    submitted = true;
    disableButtons();
    var latency = Math.round(performance.now() - renderStartTime);
    fetch('/api/attempt', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({quiz_attempt_id: attemptId, question_id: q.id, response: String(sel), latency_ms: latency})
    }).then(function(r) {
      if (!r.ok) throw new Error('Server error');
      return r.json();
    }).then(function(res) {
      showChoiceFeedback(q, sel, res);
    }).catch(function(err) {
      submitted = false;
      showError('Failed to submit. Please try again.');
    });
  }

  function submitRecall(q, isCorrect) {
    submitted = true;
    disableButtons();
    var latency = Math.round(performance.now() - renderStartTime);
    fetch('/api/attempt', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({quiz_attempt_id: attemptId, question_id: q.id, correct: isCorrect, latency_ms: latency})
    }).then(function(r) {
      if (!r.ok) throw new Error('Server error');
      return r.json();
    }).then(function(res) {
      showRecallFeedback(q, isCorrect);
    }).catch(function(err) {
      submitted = false;
      showError('Failed to submit. Please try again.');
    });
  }

  function showChoiceFeedback(q, sel, res) {
    responses[q.id] = {selected: sel, correctIndex: res.correct_index};
    var wasNew = !answeredIds[q.id];
    answeredIds[q.id] = true;
    results[q.id] = res.correct;
    if (wasNew) { answeredCount++; if (res.correct) correctCount++; }
    else {
      correctCount = 0;
      for (var id in results) { if (results[id]) correctCount++; }
    }
    var card = document.getElementById('question-area');
    var correctIdx = res.correct_index;
    var btns = card.querySelectorAll('.option-btn');
    btns.forEach(function(btn, i) {
      btn.classList.remove('hover:bg-slate-50', 'hover:border-slate-300', 'cursor-pointer');
      var letter = '<span class="w-6 h-6 rounded-full flex items-center justify-center text-xs shrink-0">' + letters[i] + '</span>';
      if (i === correctIdx) {
        btn.className = 'w-full text-left flex items-center gap-3 p-3 rounded-lg border-2 border-emerald-600 bg-emerald-100 text-sm text-slate-800 font-medium';
        btn.innerHTML = '<span class="w-6 h-6 rounded-full bg-emerald-600 text-white flex items-center justify-center text-xs shrink-0">' + letters[i] + '</span>' + q.options[i] + '<svg class="ml-auto shrink-0 text-emerald-600" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"/></svg>';
      } else if (i === sel) {
        btn.className = 'w-full text-left flex items-center gap-3 p-3 rounded-lg border-2 border-red-600 bg-red-100 text-sm text-slate-800 font-medium';
        btn.innerHTML = '<span class="w-6 h-6 rounded-full bg-red-600 text-white flex items-center justify-center text-xs shrink-0">' + letters[i] + '</span>' + q.options[i] + '<svg class="ml-auto shrink-0 text-red-600" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>';
      } else {
        btn.className = 'w-full text-left flex items-center gap-3 p-3 rounded-lg border border-slate-200 text-sm text-slate-400 line-through';
        btn.innerHTML = '<span class="w-6 h-6 rounded-full border border-slate-300 flex items-center justify-center text-xs text-slate-400 shrink-0">' + letters[i] + '</span>' + q.options[i];
      }
    });
    showNextButton();
  }

  function showRecallFeedback(q, isCorrect) {
    responses[q.id] = {isCorrect: isCorrect};
    var wasNew = !answeredIds[q.id];
    answeredIds[q.id] = true;
    results[q.id] = isCorrect;
    if (wasNew) { answeredCount++; if (isCorrect) correctCount++; }
    else {
      correctCount = 0;
      for (var id in results) { if (results[id]) correctCount++; }
    }
    var actions = document.getElementById('attempt-actions');
    if (actions) actions.innerHTML = '';
    showNextButton();
  }

  function showNextButton() {
    var actions = document.getElementById('attempt-actions');
    if (actions) actions.innerHTML = '';
    var isLast = currentIdx >= questions.length - 1;
    var advBtn = document.createElement('button');
    advBtn.id = 'advance-btn';
    advBtn.textContent = isLast ? 'Finish' : 'Next →';
    advBtn.className = 'bg-blue-700 text-white text-sm font-medium py-2 px-4 rounded-lg hover:bg-blue-700/90 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-700';
    advBtn.addEventListener('click', function() {
      if (isLast) {
        advBtn.disabled = true;
        advBtn.textContent = 'Finishing...';
        completeAttempt();
      } else {
        currentIdx++;
        renderQuestion();
      }
    });
    if (actions) actions.appendChild(advBtn);
    var scoreLine = document.getElementById('score-line');
    if (scoreLine) scoreLine.textContent = correctCount + '/' + questions.length + ' correct so far';
    updateNav();
    renderDots();
  }

  function completeAttempt() {
    fetch('/api/quiz-attempt/' + attemptId + '/complete', {method: 'POST'})
      .then(function(r) { if (!r.ok) throw new Error(); showCompletion(); })
      .catch(function() {
        var advBtn = document.getElementById('advance-btn');
        if (advBtn) { advBtn.disabled = false; advBtn.textContent = 'Finish'; }
        showError('Failed to complete. Please try again.');
      });
  }

  function showCompletion() {
    var dots = document.getElementById('attempt-dots');
    if (dots) dots.innerHTML = '';
    var actions = document.getElementById('attempt-actions');
    if (actions) actions.innerHTML = '';
    var scoreLine = document.getElementById('score-line');
    if (scoreLine) scoreLine.textContent = '';
    var navRow = document.getElementById('attempt-nav-row');
    if (navRow) navRow.style.display = 'none';
    var card = document.getElementById('question-area');
    var pct = questions.length > 0 ? Math.round(correctCount * 100 / questions.length) : 0;
    var html = '<div class="text-center py-8">';
    html += '<p class="text-3xl font-semibold text-slate-800 tracking-tight">' + correctCount + '/' + questions.length + '</p>';
    html += '<p class="text-sm text-slate-400 mt-1">' + (pct === 100 ? 'Perfect score!' : pct >= 50 ? 'Nice work.' : 'Keep practicing.') + '</p>';
    html += '</div>';
    html += '<div class="flex gap-3 justify-center mt-6">';
    html += '<a href="/workspace/' + data.workspace + '/quiz/' + data.quizSlug + '/review/' + attemptId + '" class="bg-blue-700 text-white text-sm font-medium py-2 px-4 rounded-lg hover:bg-blue-700/90 transition-colors cursor-pointer">Review answers</a>';
    html += '<a href="/workspace/' + data.workspace + '/quizzes" class="text-sm text-slate-500 hover:text-slate-700 py-2 px-4 transition-colors cursor-pointer">Back to quizzes</a>';
    html += '</div>';
    card.innerHTML = html;
  }

  window.abandonQuiz = function() {
    if (!confirm('Quit this quiz? Your progress on unanswered questions will be lost.')) return;
    fetch('/api/quiz-attempt/' + attemptId + '/abandon', {method: 'POST'})
      .then(function() { window.location.href = '/workspace/' + data.workspace + '/quizzes'; })
      .catch(function() { showError('Failed to quit. Please try again.'); });
  };

  var prevBtn = document.getElementById('attempt-prev');
  if (prevBtn) prevBtn.addEventListener('click', function() {
    if (currentIdx > 0) { currentIdx--; renderQuestion(); }
  });

  renderQuestion();
})();
`


