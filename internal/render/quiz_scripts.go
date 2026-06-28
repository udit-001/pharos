package render

// quizAttemptJS is the client-side logic for the quiz attempt page. It reads
// question data from a JSON script block, renders one card at a time, times
// each answer via performance.now(), submits via fetch, and shows inline
// feedback. No JS framework — vanilla DOM manipulation matching the site.
const quizAttemptJS = `
(function() {
  var data = JSON.parse(document.getElementById('attempt-data').textContent);
  var questions = data.questions;
  var attemptId = data.attemptId;
  var answeredIds = {};
  var results = {};
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
    var c = document.getElementById('progress-dots');
    c.innerHTML = '';
    for (var i = 0; i < questions.length; i++) {
      var d = document.createElement('span');
      d.className = 'w-5 h-1.5 rounded-full ';
      if (answeredIds[questions[i].id]) {
        d.className += results[questions[i].id] ? 'bg-emerald-600' : 'bg-red-600';
      } else if (i === currentIdx) {
        d.className += 'bg-blue-700';
      } else {
        d.className += 'bg-slate-200';
      }
      c.appendChild(d);
    }
    document.getElementById('progress-label').textContent = (currentIdx + 1) + ' of ' + questions.length;
  }

  function renderQuestion() {
    submitted = false;
    var q = questions[currentIdx];
    renderStartTime = performance.now();
    var card = document.getElementById('question-card');
    var html = '<h3 class="text-base font-medium text-slate-800 leading-relaxed">' + q.title + '</h3>';
    if (q.mode === 'choice') {
      html += '<div class="space-y-2" id="options">';
      for (var i = 0; i < q.options.length; i++) {
        html += '<button data-idx="' + i + '" class="option-btn w-full text-left flex items-center gap-3 p-3 rounded-lg border border-slate-200 hover:bg-slate-50 hover:border-slate-300 transition-colors cursor-pointer text-sm text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-700">';
        html += '<span class="w-6 h-6 rounded-full border border-slate-300 flex items-center justify-center text-xs text-slate-500 shrink-0">' + letters[i] + '</span>';
        html += q.options[i] + '</button>';
      }
      html += '</div>';
      html += '<button id="submit-btn" disabled class="w-full bg-slate-300 text-slate-500 text-sm font-medium py-2.5 px-4 rounded-lg cursor-not-allowed transition-colors">Submit answer</button>';
    } else if (q.mode === 'recall') {
      html += '<div class="p-4 rounded-lg border border-dashed border-slate-300 bg-slate-50 text-sm text-slate-700 leading-relaxed">' + q.reveal + '</div>';
      html += '<div class="flex gap-3">';
      html += '<button id="got-it-btn" class="flex-1 bg-emerald-600 text-white text-sm font-medium py-2.5 px-4 rounded-lg hover:bg-emerald-600/90 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-emerald-600">Got it</button>';
      html += '<button id="not-yet-btn" class="flex-1 bg-red-600 text-white text-sm font-medium py-2.5 px-4 rounded-lg hover:bg-red-600/90 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-red-600">Not yet</button>';
      html += '</div>';
    }
    card.innerHTML = html;

    if (q.mode === 'choice') {
      var selected = -1;
      var btns = card.querySelectorAll('.option-btn');
      btns.forEach(function(btn) {
        btn.addEventListener('click', function() {
          btns.forEach(function(b) { b.classList.remove('border-blue-700', 'bg-blue-100'); });
          btn.classList.add('border-blue-700', 'bg-blue-100');
          selected = parseInt(btn.dataset.idx);
          var sb = document.getElementById('submit-btn');
          sb.disabled = false;
          sb.className = 'w-full bg-slate-800 text-white text-sm font-medium py-2.5 px-4 rounded-lg hover:bg-slate-700 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-700';
        });
      });
      document.getElementById('submit-btn').addEventListener('click', function() {
        if (submitted || selected < 0) return;
        submitChoice(q, selected);
      });
    } else if (q.mode === 'recall') {
      document.getElementById('got-it-btn').addEventListener('click', function() {
        if (submitted) return;
        submitRecall(q, true);
      });
      document.getElementById('not-yet-btn').addEventListener('click', function() {
        if (submitted) return;
        submitRecall(q, false);
      });
    }
    renderDots();
  }

  function disableButtons() {
    var btns = document.querySelectorAll('#question-card button');
    btns.forEach(function(b) { b.disabled = true; b.style.opacity = '0.5'; });
  }

  function showError(msg) {
    var card = document.getElementById('question-card');
    var err = document.createElement('div');
    err.className = 'text-xs text-red-600';
    err.textContent = msg || 'Something went wrong. Please try again.';
    card.appendChild(err);
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
    answeredIds[q.id] = true;
    results[q.id] = res.correct;
    answeredCount++;
    if (res.correct) correctCount++;
    var card = document.getElementById('question-card');
    var correctIdx = res.correct_index;
    var btns = card.querySelectorAll('.option-btn');
    btns.forEach(function(btn, i) {
      btn.classList.remove('hover:bg-slate-50', 'hover:border-slate-300', 'cursor-pointer');
      if (i === correctIdx) {
        btn.className = 'w-full text-left flex items-center gap-3 p-3 rounded-lg border border-emerald-600 bg-emerald-100 text-sm text-slate-700';
        btn.innerHTML += '<svg class="ml-auto shrink-0" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>';
      } else if (i === sel) {
        btn.className = 'w-full text-left flex items-center gap-3 p-3 rounded-lg border border-red-600 bg-red-100 text-sm text-slate-700';
        btn.innerHTML += '<svg class="ml-auto shrink-0" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>';
      } else {
        btn.className = 'w-full text-left flex items-center gap-3 p-3 rounded-lg border border-slate-200 text-sm text-slate-400 line-through';
      }
    });
    showNextButton();
  }

  function showRecallFeedback(q, isCorrect) {
    answeredIds[q.id] = true;
    results[q.id] = isCorrect;
    answeredCount++;
    if (isCorrect) correctCount++;
    var card = document.getElementById('question-card');
    var note = document.createElement('div');
    note.className = 'text-xs ' + (isCorrect ? 'text-emerald-600' : 'text-red-600');
    note.textContent = isCorrect ? 'Marked as known.' : 'Marked for review.';
    card.appendChild(note);
    showNextButton();
  }

  function showNextButton() {
    var card = document.getElementById('question-card');
    var sb = document.createElement('button');
    var isLast = currentIdx >= questions.length - 1;
    sb.className = 'w-full bg-slate-800 text-white text-sm font-medium py-2.5 px-4 rounded-lg hover:bg-slate-700 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-700';
    sb.textContent = isLast ? 'Finish' : 'Next';
    sb.onclick = function() {
      if (isLast) {
        sb.disabled = true;
        sb.textContent = 'Finishing...';
        fetch('/api/quiz-attempt/' + attemptId + '/complete', {method: 'POST'})
          .then(function(r) { if (!r.ok) throw new Error(); window.location.href = '/workspace/' + data.workspace + '/quiz/' + data.quizSlug + '/review/' + attemptId; })
          .catch(function() { sb.disabled = false; sb.textContent = 'Finish'; showError('Failed to complete. Please try again.'); });
      } else { currentIdx++; renderQuestion(); }
    };
    card.appendChild(sb);
    var scoreLine = document.createElement('div');
    scoreLine.className = 'flex items-center justify-between text-xs text-slate-500';
    scoreLine.innerHTML = '<span>' + correctCount + '/' + answeredCount + ' correct so far</span>';
    card.appendChild(scoreLine);
    renderDots();
  }

  window.abandonQuiz = function() {
    if (!confirm('Quit this quiz? Your progress on unanswered questions will be lost.')) return;
    fetch('/api/quiz-attempt/' + attemptId + '/abandon', {method: 'POST'})
      .then(function() { window.location.href = '/workspace/' + data.workspace + '/quizzes'; })
      .catch(function() { showError('Failed to quit. Please try again.'); });
  };

  renderQuestion();
})();
`

// quizReviewJS is the client-side logic for the quiz review page. It renders
// one review card at a time with prev/next nav and clickable progress dots.
const quizReviewJS = `
(function() {
  var data = JSON.parse(document.getElementById('review-data').textContent);
  var items = data.items;
  var idx = 0;

  function renderDots() {
    var c = document.getElementById('review-dots');
    c.innerHTML = '';
    for (var i = 0; i < items.length; i++) {
      var d = document.createElement('span');
      d.className = 'w-5 h-1.5 rounded-full cursor-pointer hover:ring-2 hover:ring-slate-300 ' + (items[i].IsCorrect ? 'bg-emerald-600' : 'bg-red-600');
      if (i === idx) d.className += ' ring-2 ring-slate-400';
      d.onclick = (function(n) { return function() { idx = n; render(); }; })(i);
      c.appendChild(d);
    }
    document.getElementById('progress-label').textContent = 'Question ' + (idx + 1) + ' of ' + items.length;
  }

  function render() {
    var item = items[idx];
    var card = document.getElementById('review-card');
    var html = '<div class="space-y-4 p-4 rounded-lg border border-slate-200">';
    html += '<div class="flex items-center justify-end">';
    html += '<span class="inline-flex items-center ' + (item.IsCorrect ? 'bg-emerald-100 text-emerald-600' : 'bg-red-100 text-red-600') + ' text-xs font-medium px-2 py-0.5 rounded">' + (item.IsCorrect ? 'Correct' : 'Incorrect') + '</span>';
    html += '</div>';
    html += '<h4 class="text-sm font-medium text-slate-800 leading-relaxed">' + item.QuestionTitle + '</h4>';
    if (item.Mode === 'choice') {
      var userResp = parseInt(item.UserResponse);
      for (var i = 0; i < item.Options.length; i++) {
        if (i === item.CorrectIndex) {
          html += '<div class="flex items-center gap-2 p-2 rounded border border-emerald-600 bg-emerald-100 text-xs"><span class="text-emerald-600 font-medium">Correct:</span>' + item.Options[i] + '</div>';
        } else if (i === userResp) {
          html += '<div class="flex items-center gap-2 p-2 rounded border border-red-600 bg-red-100 text-xs"><span class="text-red-600 font-medium">You answered:</span>' + item.Options[i] + '</div>';
        } else {
          html += '<div class="flex items-center gap-2 p-2 rounded border border-slate-200 text-xs text-slate-400">' + item.Options[i] + '</div>';
        }
      }
    } else if (item.Mode === 'recall') {
      html += '<div class="p-2 rounded bg-slate-50 text-xs text-slate-500 leading-relaxed">' + item.RevealText + '</div>';
      html += '<div class="text-xs ' + (item.IsCorrect ? 'text-emerald-600' : 'text-red-600') + '">' + (item.IsCorrect ? 'You marked this as known.' : 'You marked this for review.') + '</div>';
    }
    html += '</div>';
    card.innerHTML = html;

    var nav = document.getElementById('review-nav');
    nav.innerHTML = '';
    var prevB = document.createElement('button');
    prevB.className = 'flex items-center gap-1 text-sm text-slate-500 hover:text-slate-800 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-700 rounded' + (idx === 0 ? ' opacity-30 cursor-not-allowed' : '');
    prevB.textContent = 'Previous';
    if (idx > 0) prevB.onclick = function() { idx--; render(); };
    nav.appendChild(prevB);
    var nextB = document.createElement('button');
    nextB.className = 'flex items-center gap-1 text-sm text-slate-500 hover:text-slate-800 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-700 rounded' + (idx >= items.length - 1 ? ' opacity-30 cursor-not-allowed' : '');
    nextB.textContent = 'Next';
    if (idx < items.length - 1) nextB.onclick = function() { idx++; render(); };
    nav.appendChild(nextB);
    renderDots();
  }

  render();
})();
`
