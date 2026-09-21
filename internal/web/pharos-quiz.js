/**
 * pharos-quiz.js — inline quiz binder for lessons.
 *
 * Auto-injected by the server when lesson HTML contains .q blocks. Lessons
 * never include this file or copy the binder by hand — write the markup and
 * the behavior arrives:
 *
 *   <div class="q" data-answer="Bar chart">
 *     <p>Which chart compares quarterly revenue?</p>
 *     <div class="options"><button>Bar chart</button>...</div>
 *     <div class="fb"></div>
 *   </div>
 *
 * Styles (.q, .options, .fb, correct/incorrect states) live in style.css.
 *
 * Idempotent by design: the server may inject this alongside a legacy lesson
 * that still carries its own inline copy of the binder. The per-element guard
 * ensures each button ever gets one handler regardless of how many copies run.
 */
(function () {
  'use strict';

  function bind(q) {
    var answer = q.getAttribute('data-answer');
    var buttons = q.querySelectorAll('button');
    var fb = q.querySelector('.fb');
    buttons.forEach(function (btn) {
      if (btn.dataset.pharosQuizBound) return;
      btn.dataset.pharosQuizBound = '1';
      btn.addEventListener('click', function () {
        buttons.forEach(function (b) { b.disabled = true; });
        if (btn.textContent.trim() === answer) {
          btn.classList.add('correct');
          if (fb) fb.textContent = 'Correct.';
        } else {
          btn.classList.add('incorrect');
          buttons.forEach(function (b) {
            if (b.textContent.trim() === answer) b.classList.add('correct');
          });
          if (fb) fb.textContent = 'Not quite — the right one is highlighted.';
        }
      });
    });
  }

  function init() {
    if (window.__pharosQuizBound) return;
    window.__pharosQuizBound = true;
    document.querySelectorAll('.q').forEach(bind);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
