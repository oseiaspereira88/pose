"""Observe residual-work semantics in disposable Git fixtures; no network.
Run: python3 THIS_FILE --binary /absolute/path/pose --output /tmp/front05.json
Results describe behavior; they do not require defects to survive a correction.
"""
import argparse, hashlib, json, os, pathlib, subprocess, tempfile
p=argparse.ArgumentParser(description=__doc__)
p.add_argument('--binary', required=True); p.add_argument('--output', required=True)
a=p.parse_args(); binary=pathlib.Path(a.binary).resolve()
parent=pathlib.Path(tempfile.mkdtemp(prefix='pose-residual-probes-'))
env={k:v for k,v in os.environ.items() if not k.startswith(('POSE_', 'HARNE8_'))}
env['POSE_FOLLOWUP_TODAY']='2026-10-12'
def write(r,path,text):
 f=r/path; f.parent.mkdir(parents=True,exist_ok=True); f.write_text(text)
def spec(r,slug,items,status='done',extra=''):
 write(r,f'.pose/specs/{slug}.md',f'---\nslug: {slug}\nstatus: {status}\ncreated_at: 2026-10-01\ncompleted_at: 2026-10-02\n---\n\n# Spec: {slug}\n\n{extra}\n## 7. Final Report\n\n### Follow-ups\n\n'+items+'\n')
def fixture(name):
 r=parent/name; r.mkdir(); write(r,'.pose/project.json','{"schema_version":1,"project_id":"proj.residual-probe"}')
 write(r,'.pose/policy/review.json','{"schema_version":2,"enabled":false}')
 subprocess.run(['git','init','-q',str(r)],check=True)
 subprocess.run(['git','-C',str(r),'add','-A'],check=True)
 subprocess.run(['git','-C',str(r),'-c','user.name=Probe','-c','user.email=probe@example.invalid','-c','commit.gpgsign=false','commit','-qm','fixture'],check=True)
 return r
def run(r,*args):
 q=subprocess.run([str(binary),*args],cwd=r,env=env,capture_output=True,text=True,timeout=60)
 if q.returncode: raise RuntimeError(str(args)+q.stdout+q.stderr)
 return json.loads(q.stdout)
results={}
r=fixture('unmarked-and-typed'); spec(r,'alpha','- Missing mandatory correction (owner:@core crit:high review:2020-01-01)\n- [open] Confirmed release regression (owner:@core crit:high review:2020-01-01)')
f=run(r,'followups','--json'); c=run(r,'followups','--candidates','--json'); at=run(r,'state','--attention','--json')
results['P01']={'followups':f,'candidates':c,'attention':at}
r=fixture('same-file-coverage'); write(r,'pkg/sample.go','package pkg\n// TODO: repair lost writes\n// FIXME: stop corrupting output\nfunc Safe() { panic("invariant") }\n')
spec(r,'coverage','',status='draft',extra='## 1. Intent\nThis documentation task merely lists pkg/sample.go as background; it commits to no repair.')
results['P02']=run(r,'assess','tech-debt','--json')
r=fixture('repeated-marker'); write(r,'pkg/sample.go','package pkg\n// TODO: repair\n// TODO: repair\n')
results['P03']=run(r,'assess','tech-debt','--json')
r=fixture('scanner-truncation'); write(r,'pkg/sample.go','package pkg\n'+('// '+('x'*70000))+'\n// TODO: lost after long line\n')
results['P04']=run(r,'assess','tech-debt','--json')
r=fixture('invalid-date'); spec(r,'alpha','- [open] Repair (owner:@core crit:high review:2026-99-99)')
results['P05']={'followups':run(r,'followups','--json'),'candidates':run(r,'followups','--candidates','--json')}
r=fixture('target-routing'); spec(r,'alpha','- [spawned: beta] Required repair (owner:@core crit:high review:2020-01-01)'); spec(r,'beta','',status='abandoned')
results['P06']={'followups':run(r,'followups','--json'),'candidates':run(r,'followups','--candidates','--json'),'attention':run(r,'state','--attention','--json')}
r=fixture('ordinal-identity'); spec(r,'alpha','- [open] Retained repair (owner:@core crit:low review:2099-01-01)')
one=run(r,'state','--attention','--json'); spec(r,'alpha','- [open] New first item (owner:@core crit:low review:2099-01-01)\n- [open] Retained repair (owner:@core crit:low review:2099-01-01)')
two=run(r,'state','--attention','--json'); results['P07']={'before':one,'after':two}
r=fixture('candidate-no-write'); spec(r,'alpha','- [open] Check `beta` and TestExisting (owner:@core crit:low review:2099-01-01)'); spec(r,'beta','',status='done'); write(r,'pkg/sample_test.go','package pkg\nfunc TestExisting(t *testing.T) {}\n')
before=(r/'.pose/specs/alpha.md').read_bytes(); result=run(r,'followups','--candidates','--json'); after=(r/'.pose/specs/alpha.md').read_bytes()
results['P08']={'unchanged':before==after,'candidates':result}
output={'binary_sha256':hashlib.sha256(binary.read_bytes()).hexdigest(),'version':subprocess.run([str(binary),'version'],capture_output=True,text=True).stdout,'fixtures':str(parent),'observations':results}
pathlib.Path(a.output).write_text(json.dumps(output,indent=2)+'\n'); print(json.dumps({'output':a.output,'probes':len(results),'fixtures':str(parent)}))
