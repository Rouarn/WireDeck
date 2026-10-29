import * as THREE from './assets/three.module.js'

const canvas = document.querySelector('#hero-canvas')
const wrap = canvas.parentElement

const renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true })
renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2))

const scene = new THREE.Scene()
const camera = new THREE.PerspectiveCamera(42, 1, 0.1, 100)
camera.position.set(0, 0, 7.2)

const group = new THREE.Group()
scene.add(group)

// 外层线框球
const sphere = new THREE.Mesh(
  new THREE.IcosahedronGeometry(2.1, 1),
  new THREE.MeshBasicMaterial({
    color: 0xbd34fe,
    wireframe: true,
    transparent: true,
    opacity: 0.26,
  }),
)
group.add(sphere)

// 内层实体球（低透明度，制造体积感）
const inner = new THREE.Mesh(
  new THREE.IcosahedronGeometry(1.45, 0),
  new THREE.MeshBasicMaterial({
    color: 0x41d1ff,
    transparent: true,
    opacity: 0.1,
  }),
)
group.add(inner)

// 粒子云
const count = 720
const positions = new Float32Array(count * 3)
for (let i = 0; i < count; i += 1) {
  const r = 1.55 + Math.random() * 1.05
  const theta = Math.random() * Math.PI * 2
  const phi = Math.acos(2 * Math.random() - 1)
  positions[i * 3] = r * Math.sin(phi) * Math.cos(theta)
  positions[i * 3 + 1] = r * Math.sin(phi) * Math.sin(theta)
  positions[i * 3 + 2] = r * Math.cos(phi)
}
const particleGeo = new THREE.BufferGeometry()
particleGeo.setAttribute('position', new THREE.BufferAttribute(positions, 3))
const particles = new THREE.Points(
  particleGeo,
  new THREE.PointsMaterial({
    color: 0xd8b4fe,
    size: 0.035,
    transparent: true,
    opacity: 0.85,
    sizeAttenuation: true,
  }),
)
group.add(particles)

// 两个环绕的细环
const ring1 = new THREE.Mesh(
  new THREE.TorusGeometry(2.65, 0.006, 8, 140),
  new THREE.MeshBasicMaterial({ color: 0xc084fc, transparent: true, opacity: 0.55 }),
)
ring1.rotation.x = Math.PI / 2.35
group.add(ring1)

const ring2 = new THREE.Mesh(
  new THREE.TorusGeometry(2.95, 0.005, 8, 140),
  new THREE.MeshBasicMaterial({ color: 0x41d1ff, transparent: true, opacity: 0.4 }),
)
ring2.rotation.x = -Math.PI / 3
ring2.rotation.y = 0.7
group.add(ring2)

function resize() {
  const width = wrap.clientWidth
  const height = wrap.clientHeight
  renderer.setSize(width, height, false)
  camera.aspect = width / height
  camera.updateProjectionMatrix()
}

window.addEventListener('resize', resize)
resize()

let mx = 0
let my = 0
window.addEventListener('pointermove', (event) => {
  mx = event.clientX / window.innerWidth - 0.5
  my = event.clientY / window.innerHeight - 0.5
})

const clock = new THREE.Clock()

function animate() {
  const t = clock.getElapsedTime()
  group.rotation.y = t * 0.22 + mx * 0.8
  group.rotation.x = Math.sin(t * 0.18) * 0.12 + my * 0.45
  ring1.rotation.z = t * 0.3
  ring2.rotation.z = -t * 0.22
  particles.rotation.y = t * 0.05
  renderer.render(scene, camera)
  requestAnimationFrame(animate)
}

animate()

// 为每个功能卡片插入一个缓慢旋转的 3D 方块，增加页面动感。
document.querySelectorAll('.visual').forEach((el) => {
  const cube = document.createElement('div')
  cube.className = 'cube'
  cube.innerHTML = '<i></i>'.repeat(6)
  el.appendChild(cube)
})
