"""Synthetic Unix probes for POSE 7.4.1 dispatch; never runs vendor agents.

Run with --binary /absolute/path/to/pose --output /tmp/front03.json.
All mutations, including the outside-worktree probe, affect new temp fixtures.
Fixtures are preserved under the printed directory for manual inspection.
Concurrent-budget results depend on scheduling; repeat if processes serialize.
The fixture's delivery graph is synthetic, not validation of a production module.
"""
import argparse, concurrent.futures, json, os, pathlib, shlex, shutil, subprocess, tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--binary', required=True)
parser.add_argument('--output', required=True)
parser.add_argument('--scratch-root', default=tempfile.gettempdir())
opts = parser.parse_args()

scratch=pathlib.Path(opts.scratch_root).resolve()
binary=pathlib.Path(opts.binary).resolve()
if os.name != 'posix' or not shutil.which('sh') or not shutil.which('git'):
    raise SystemExit('These probes require Unix, sh and git.')
if not binary.is_file():
    raise SystemExit('The --binary file does not exist.')
parent=pathlib.Path(tempfile.mkdtemp(prefix='pose-dispatch-probes-',dir=scratch))
base_env={k:v for k,v in os.environ.items() if not k.startswith(('POSE_','HARNE8_'))}
actual_git=shutil.which('git')

def write(root,path,content):
    p=root/path;p.parent.mkdir(parents=True,exist_ok=True)
    p.write_text(json.dumps(content) if isinstance(content,dict) else content)

def invoke(root,args,env=None):
    r=subprocess.run([str(binary),*args],cwd=root,env=env or base_env,capture_output=True,text=True,timeout=30)
    return r

def fixture(name):
    root=parent/name;root.mkdir()
    write(root,'.pose/project.json',{'schema_version':1,'project_id':'proj.dispatch-probe'})
    write(root,'.pose/policy/review.json',{'schema_version':2,'enabled':True,'adopted_at':'2026-08-02','profiles':{'spec':'spec-closeout@1'},'review_bundles':True,'review_bundles_adopted_at':'2026-08-14'})
    write(root,'.pose/review-profiles/spec-closeout.json',{'schema_version':1,'id':'spec-closeout','version':1,'scope':'spec','criteria':[{'id':'correctness','description':'reviewed'}]})
    write(root,'.pose/specs/alpha/spec.md','---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\ncompleted_at:\n---\n\n# Spec: alpha\n\n## 2. Requirements\n- R1: works\n\n## 6. Validation\n\n### Requirement trace\n- R1 [satisfied] test:TestFixture\n')
    write(root,'pose-mcp/lib.go','package posemcp\n')
    def git(*args):return subprocess.check_output([actual_git,'-C',str(root),'-c','user.name=Probe','-c','user.email=probe@example.invalid','-c','commit.gpgsign=false',*args],stderr=subprocess.DEVNULL,text=True).strip()
    git('init','-q');git('add','-A');git('commit','-q','-m','alpha');head=git('rev-parse','HEAD')
    prov='sha256:'+'1'*64
    graph={'schema_version':1,'input_digest':'','provenance_digest':prov,'nodes':None,'edges':None,'claims':None,'change_sets':[{'id':'cs-alpha','spec':'alpha','selector':'range:'+head+'..'+head,'base':head,'head':head,'resolved_base':head,'resolved_head':head,'paths':[{'action':'modified','path':'pose-mcp/lib.go'}],'diff_digest':'sha256:'+'2'*64}],'reverse':{'pose-mcp/lib.go':['alpha']},'findings':None,'deliveries':[{'spec':'alpha','ref':'contract:alpha-api','kind':'contract','id':'alpha-api','module':'pose-mcp','profile':'api-contract','entrypoint':'pose-mcp/lib.go'}],'validation_results':[{'id':'val-alpha','module':'pose-mcp','check':'go-test','evidence_class':'integration','severity':'required','outcome':'pass','git_head':head,'provenance_digest':prov}]}
    write(root,'.pose/indexes/delivery-integrity.json',graph)
    r=invoke(root,['review','bundle','spec:alpha','--seal'])
    if r.returncode:raise RuntimeError('seal failed: '+r.stdout+r.stderr)
    return root

def configure(root,command,budget=1,timeout='5s',name='probe'):
    write(root,'.pose/policy/reviewers.json',{'schema_version':1,'adapters':{name:{'command':command,'vendor':'fake','model':'fixture-1','timeout':timeout,'max_runs_per_bundle':budget}}})

def runs(root):return [json.loads(p.read_text()) for p in sorted((root/'.pose/review-runs').glob('*.json'))]

def dispatch(root,via='probe',env=None):return invoke(root,['review','dispatch','spec:alpha','--via',via,'--apply'],env)

results={}
root=fixture('empty-output');configure(root,['sh','-c','cat >/dev/null; true'])
r=dispatch(root);run=runs(root)[-1]
results['empty_output']={'exit_code':r.returncode,'run_status':run['status'],'output_empty':not run['output'],'attestations_written':len(list((root/'.pose/review-attestations').glob('*.json')))}

root=fixture('budget');configure(root,['sh','-c','cat >/dev/null; sleep 0.6; echo decision: changes-requested'])
with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
    out=list(pool.map(lambda _:dispatch(root),range(2)))
rr=runs(root)
results['concurrent_budget']={'max_runs_per_bundle':1,'exit_codes':[r.returncode for r in out],'completed_runs':sum(r['status']=='completed' for r in rr),'recorded_runs':len(rr)}

root=fixture('edited-content');configure(root,['sh','-c','cat >/dev/null; echo changed >> pose-mcp/lib.go'])
r=dispatch(root);run=runs(root)[-1]
results['edit_control']={'exit_code':r.returncode,'run_status':run['status'],'failure':run.get('failure')}

root=fixture('assume-unchanged');configure(root,['sh','-c','cat >/dev/null; git update-index --assume-unchanged pose-mcp/lib.go; echo changed >> pose-mcp/lib.go; echo decision: approved'])
r=dispatch(root);run=runs(root)[-1]
results['assume_unchanged_edit']={'exit_code':r.returncode,'run_status':run['status'],'failure':run.get('failure'),'attestations_written':len(list((root/'.pose/review-attestations').glob('*.json')))}

root=fixture('status-error');flag=parent/'status-error.flag';shim=parent/'git-shim';shim.mkdir()
code='#!/usr/bin/env python3\nimport os,sys,pathlib\na=sys.argv[1:]\nif "status" in a and "-C" in a and pathlib.Path(a[a.index("-C")+1]).name=="worktree" and pathlib.Path('+repr(str(flag))+').exists():sys.exit(31)\nos.execv('+repr(actual_git)+',["git",*a])\n'
write(shim,'git',code);(shim/'git').chmod(0o755)
configure(root,['sh','-c','cat >/dev/null; echo changed >> pose-mcp/lib.go; touch '+shlex.quote(str(flag))])
env={**base_env,'PATH':str(shim)+os.pathsep+base_env['PATH']}
r=dispatch(root,env=env);run=runs(root)[-1]
results['post_run_git_status_error']={'exit_code':r.returncode,'run_status':run['status'],'failure':run.get('failure')}

root=fixture('outside-write');outside=root/'outside-proof.txt';configure(root,['sh','-c','cat >/dev/null; echo fixture-change > '+shlex.quote(str(outside))+'; echo decision: approved'])
r=dispatch(root);run=runs(root)[-1]
results['main_fixture_write']={'exit_code':r.returncode,'run_status':run['status'],'outside_worktree_changed':outside.exists()}

# A changed record and transcript are not validated when budget is counted.
root=fixture('ledger');configure(root,['sh','-c','cat >/dev/null; echo decision: changes-requested'])
r=dispatch(root);record=list((root/'.pose/review-runs').glob('*.json'))[0];data=json.loads(record.read_text());transcript=root/data['transcript_path'];transcript.write_text('tampered fixture transcript\n');data['adapter']='other';data['output']='decision: approved';record.write_text(json.dumps(data))
r=dispatch(root);results['tampered_budget_record']={'exit_code':r.returncode,'completed_runs':sum(x['status']=='completed' for x in runs(root)),'records_written':len(runs(root)),'tampered_transcript_present':transcript.exists()}

# Rejections and changes-requested are merely output text today.
root=fixture('rerun');configure(root,['sh','-c','cat >/dev/null; echo decision: changes-requested'],budget=2)
a=dispatch(root);b=dispatch(root);results['unchanged_rerun']={'exit_codes':[a.returncode,b.returncode],'run_statuses':[x['status'] for x in runs(root)],'same_bundle':len({x['bundle_id'] for x in runs(root)})==1,'same_brief':len({x['brief_digest'] for x in runs(root)})==1}

root=fixture('timeout');configure(root,['sh','-c','cat >/dev/null; sleep 5'],timeout='100ms')
r=dispatch(root);run=runs(root)[-1];results['timeout_control']={'exit_code':r.returncode,'run_status':run['status'],'failure':run.get('failure')}

root=fixture('setup-failure');configure(root,['sh','-c','cat >/dev/null; true'])
# Deny worktree creation only after planning; no reviewer starts.
shim2=parent/'git-worktree-shim';shim2.mkdir()
write(shim2,'git','#!/usr/bin/env python3\nimport os,sys\na=sys.argv[1:]\nif "worktree" in a and "add" in a:sys.exit(32)\nos.execv('+repr(actual_git)+',["git",*a])\n');(shim2/'git').chmod(0o755)
r=dispatch(root,env={**base_env,'PATH':str(shim2)+os.pathsep+base_env['PATH']})
results['setup_failure_record']={'exit_code':r.returncode,'recorded_runs':len(runs(root)),'stderr':r.stderr.strip()[-300:]}

pathlib.Path(opts.output).write_text(json.dumps(results,ensure_ascii=False,indent=2)+'\n')
print(json.dumps(results,ensure_ascii=False,indent=2))
print('Fixtures: ' + str(parent))
