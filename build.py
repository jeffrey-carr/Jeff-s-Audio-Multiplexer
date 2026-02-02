import os
import subprocess
import shutil

if __name__ == "__main__":
  print('building program...')

  if os.listdir('.').count('output') > 0:
    for filename in os.listdir('output'):
      print(f'Removing output/{filename}...')
      os.remove(os.path.join('output', filename))
  else:
    os.mkdir('output')

  subprocess.run(['go', 'build', '-o', 'output/', './...'])
  shutil.copyfile('config.yaml', 'output/config.yaml')