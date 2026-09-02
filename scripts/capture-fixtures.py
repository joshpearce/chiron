import json, urllib.request, sys, os
B = sys.argv[1] if len(sys.argv) > 1 else 'http://localhost:8084'
OUT = sys.argv[2] if len(sys.argv) > 2 else 'fixtures'
def get(p):
    return json.load(urllib.request.urlopen(B + p, timeout=600))
def post(p, body):
    r = urllib.request.Request(B + p, data=json.dumps(body).encode(), headers={'Content-Type': 'application/json'})
    return json.load(urllib.request.urlopen(r, timeout=600))
def save(name, obj):
    with open(os.path.join(OUT, name + '.json'), 'w') as f:
        json.dump(obj, f, indent=1, sort_keys=True); f.write('\n')
    print('saved', name, list(obj.keys()) if isinstance(obj, dict) else type(obj))

def walk(subject, save_as):
    start = post('/exchange', {'subject': subject, 'phase': 'start'})
    if save_as: save('exchange-start', start)
    ch = start['chapter']; scr = ch['check'][0]; assert scr['check'] == 'screener', scr
    placed = post('/exchange', {'subject': subject, 'phase': 'boundary', 'unit': ch['unit'],
                                'check_responses': [{'item_id': scr['id'], 'selected_index': 2, 'confidence': 3}]})
    if save_as: save('exchange-screener', placed)
    series = placed['chapter']; assert series.get('calibration') and len(series['check']) > 1
    resp = []
    for it in series['check']:
        if it['kind'] == 'mcq':
            idx = next(i for i, o in enumerate(it['reveal']['options']) if o['correct'])
            resp.append({'item_id': it['id'], 'selected_index': idx, 'confidence': 4})
        elif it['check'] == 'llm':
            resp.append({'item_id': it['id'], 'idk': True, 'confidence': 1})
        else:
            resp.append({'item_id': it['id'], 'response': it['reveal']['answer'], 'confidence': 4})
    graded = post('/exchange', {'subject': subject, 'phase': 'boundary', 'unit': series['unit'], 'check_responses': resp, 'chunk_minutes': 3.5})
    print(subject, 'gate', graded.get('gate'), 'chapter', (graded.get('chapter') or {}).get('unit'), 'extra keys', [k for k in graded if k not in ('results','gate','chapter','state','break_suggestion')])
    if save_as: save('exchange-series', graded)

walk('ai', True)
walk('data', False)
save('subjects', get('/subjects'))
save('state', get('/state?subject=ai'))
for s in ('ai', 'data'):
    post('/reset', {'subject': s, 'confirm': True})
print('reset both; active now', get('/subjects')['active'])
