#!/usr/bin/env python3
"""Partition the complete HTTP consumer without dropping source or assertions."""
from collections import defaultdict
import heapq
import json
from pathlib import Path
import re
import shutil
import sys

# Group independently owned packages before packing. A connected component may
# exceed this target; it stays intact rather than weakening any width/depth probe.
TARGET_BYTES = 384 * 1024
CONTRACT_PACKAGE = 'org.example.contracts'
PACKAGE = re.compile(r'^package\s+([\w.]+)', re.MULTILINE)
QUALIFIED = re.compile(r'\borg\.example\.[\w.]+')
DECLARATION = re.compile(
    r'^((?:(?:private|internal|public|suspend|inline|data|sealed|enum|value|open|abstract|actual|expect)[ \t]+)*)'
    r'(?:fun|class|object|interface|typealias)[ \t]+([A-Za-z_]\w*)', re.MULTILINE)


class Groups:
    def __init__(self, keys):
        self.parent = {key: key for key in keys}

    def find(self, key):
        while self.parent[key] != key:
            self.parent[key] = self.parent[self.parent[key]]
            key = self.parent[key]
        return key

    def join(self, first, second):
        first, second = self.find(first), self.find(second)
        if first != second:
            self.parent[max(first, second)] = min(first, second)


def pack(groups, sizes, ordered=False):
    result, current, used = [], [], 0
    groups = [sorted(group) for group in groups]
    if not ordered:
        groups.sort(key=lambda group: group[0])
    for group in groups:
        size = sum(sizes[path] for path in group)
        if current and used + size > TARGET_BYTES:
            result.append(current)
            current, used = [], 0
        current.extend(group)
        used += size
    if current:
        result.append(current)
    return result


def dependency_components(edges):
    """Return actual recursive components in deterministic dependency order."""
    seen, finished = set(), []
    for source in sorted(edges):
        stack = [(source, False)]
        while stack:
            node, leaving = stack.pop()
            if leaving:
                finished.append(node)
            elif node not in seen:
                seen.add(node)
                stack.append((node, True))
                stack.extend((child, False) for child in sorted(edges[node], reverse=True) if child not in seen)
    incoming = defaultdict(set)
    for node, children in edges.items():
        for child in children:
            incoming[child].add(node)
    owners, components = {}, []
    for source in reversed(finished):
        if source in owners:
            continue
        component = []
        stack = [source]
        index = len(components)
        while stack:
            node = stack.pop()
            if node in owners:
                continue
            owners[node] = index
            component.append(node)
            stack.extend(sorted(incoming[node], reverse=True))
        components.append(sorted(component))
    dependencies = {i: set() for i in range(len(components))}
    parents = defaultdict(set)
    for node, children in edges.items():
        for child in children:
            if owners[node] != owners[child]:
                dependencies[owners[node]].add(owners[child])
                parents[owners[child]].add(owners[node])
    ready = [(component[0], i) for i, component in enumerate(components) if not dependencies[i]]
    heapq.heapify(ready)
    result = []
    while ready:
        _, index = heapq.heappop(ready)
        result.append(components[index])
        for parent in sorted(parents[index]):
            dependencies[parent].remove(index)
            if not dependencies[parent]:
                heapq.heappush(ready, (components[parent][0], parent))
    if len(result) != len(components):
        raise ValueError('Recursive source components must form a dependency DAG')
    return result


def partition(project):
    project = Path(project).resolve()
    sources = {}
    for directory in sorted((project / 'build').glob('generated*')):
        for path in sorted(directory.rglob('*.kt')):
            sources[path.relative_to(project).as_posix()] = path.read_text()
    for path in sorted((project / 'src/main/kotlin').rglob('*.kt')):
        sources[path.relative_to(project).as_posix()] = path.read_text()
    runner = 'src/main/kotlin/org/example/contracts/HttpContracts.kt'
    if runner not in sources:
        raise ValueError('The unchanged complete HTTP runtime registry is required')
    packages = {}
    for path, text in sources.items():
        match = PACKAGE.search(text)
        if not match:
            raise ValueError('Every Kotlin source needs an explicit package: ' + path)
        packages[path] = match[1]
    sizes = {path: len(text.encode()) for path, text in sources.items()}
    libraries = [path for path in sources if packages[path] != CONTRACT_PACKAGE]
    contracts = [path for path in sources if packages[path] == CONTRACT_PACKAGE and path != runner]
    symbols = defaultdict(set)
    local_symbols = defaultdict(lambda: defaultdict(set))
    for path in libraries:
        names = {Path(path).stem}
        code = re.sub(r'""".*?"""', '', sources[path], flags=re.DOTALL)
        for match in DECLARATION.finditer(code):
            if 'private' not in match[1].split():
                names.add(match[2])
        for name in names:
            symbols[packages[path] + '.' + name].add(path)
            local_symbols[packages[path]][name].add(path)
    edges = {path: set() for path in libraries}
    for path in libraries:
        for name in QUALIFIED.findall(sources[path]):
            while name:
                if name in symbols:
                    edges[path].update(symbols[name])
                    break
                name = name.rpartition('.')[0]
        for name in set(re.findall(r'\b[A-Za-z_]\w*\b', sources[path])):
            edges[path].update(local_symbols[packages[path]].get(name, []))
        edges[path].discard(path)
    library_shards = pack(dependency_components(edges), sizes, ordered=True)

    # Keep shared internal test engines/helpers in their consuming compilation.
    # Raw test-data strings must not invent declarations (e.g. a fixture main).
    exports = defaultdict(list)
    for path in contracts:
        code = re.sub(r'""".*?"""', '', sources[path], flags=re.DOTALL)
        for match in DECLARATION.finditer(code):
            if 'private' not in match[1].split():
                exports[match[2]].append(path)
    groups = Groups(contracts)
    for path in contracts:
        for name in set(re.findall(r'\b[A-Za-z_]\w*\b', sources[path])):
            for owner in exports.get(name, []):
                groups.join(path, owner)
    components = defaultdict(list)
    for path in contracts:
        components[groups.find(path)].append(path)
    contract_shards = pack(components.values(), sizes)

    output = project / 'build/http-compilations'
    if output.exists():
        raise ValueError('Expected a fresh HTTP compilation plan')
    output.mkdir()
    inventory = {}
    names = []
    for kind, shards in [('Library', library_shards), ('Contract', contract_shards)]:
        for index, paths in enumerate(shards):
            name = f'http{kind}{index:03d}'
            names.append((name, kind))
            inventory[name] = sorted(paths)
            for number, path in enumerate(sorted(paths)):
                target = output / name / str(number) / Path(path).name
                target.parent.mkdir(parents=True)
                shutil.copyfile(project / path, target)
    inventory['main'] = [runner]
    target = output / 'main/HttpContracts.kt'
    target.parent.mkdir()
    shutil.copyfile(project / runner, target)
    assigned = [path for paths in inventory.values() for path in paths]
    if len(assigned) != len(sources) or set(assigned) != set(sources):
        raise ValueError('Every original source must belong to exactly one compilation')
    (output / 'inventory.json').write_text(json.dumps(inventory, indent=2) + '\n')
    literal = lambda values: ', '.join(json.dumps(value) for value in values)
    build = project / 'build.gradle.kts'
    build.write_text(build.read_text() + '''
// Complete HTTP source inventory, split into independent compiler invocations.
val httpLibraries = listOf(''' + literal([n for n, k in names if k == 'Library']) + ''')
val httpContracts = listOf(''' + literal([n for n, k in names if k == 'Contract']) + ''')
val httpCompilations = httpLibraries + httpContracts
// Keep declared external dependencies independent of main's friend-module
// inheritance; associateWith adds the consumer configurations back to main.
val httpDeclaredImplementation = configurations.create("httpDeclaredImplementation") {
    dependencies.addAll(configurations.implementation.get().allDependencies)
}
val httpDeclaredRuntime = configurations.create("httpDeclaredRuntime") {
    dependencies.addAll(configurations.runtimeOnly.get().allDependencies)
}
httpCompilations.forEach { name ->
    val shard = sourceSets.create(name)
    kotlin.sourceSets.getByName(name).kotlin.setSrcDirs(listOf("build/http-compilations/$name"))
    configurations.getByName(shard.implementationConfigurationName).extendsFrom(httpDeclaredImplementation)
    configurations.getByName(shard.runtimeOnlyConfigurationName).extendsFrom(httpDeclaredRuntime)
    if (name in httpContracts) {
        shard.compileClasspath += files(httpLibraries.map { sourceSets.getByName(it).output })
        shard.runtimeClasspath += files(httpLibraries.map { sourceSets.getByName(it).output })
    } else {
        val predecessors = httpLibraries.takeWhile { it != name }
        shard.compileClasspath += files(predecessors.map { sourceSets.getByName(it).output })
        shard.runtimeClasspath += files(predecessors.map { sourceSets.getByName(it).output })
    }
}
kotlin.sourceSets.getByName("main").kotlin.setSrcDirs(listOf("build/http-compilations/main"))
sourceSets.main {
    compileClasspath += files(httpCompilations.map { sourceSets.getByName(it).output })
    runtimeClasspath += files(httpCompilations.map { sourceSets.getByName(it).output })
}
// The original runtime registry can use internal helpers from its test modules.
httpContracts.forEach { name ->
    kotlin.target.compilations.getByName("main").associateWith(kotlin.target.compilations.getByName(name))
}
// Bound concurrent compiler processes without changing heap or schema limits.
var httpPrevious: String? = null
httpCompilations.forEach { name ->
    val compile = kotlin.target.compilations.getByName(name).compileTaskProvider
    httpPrevious?.let { previous -> compile.configure { mustRunAfter(previous) } }
    httpPrevious = compile.name
}
''')
    print(f'Complete HTTP compilation inventory: {len(libraries)} library/provider sources in {len(library_shards)} compilations; '
          f'{len(contracts)} contract sources in {len(contract_shards)} compilations; unchanged runtime registry')
    return inventory


if __name__ == '__main__':
    if len(sys.argv) != 2:
        raise SystemExit('Usage: python3 scripts/kotlin/http_compilations.py FRESH_PROJECT')
    partition(sys.argv[1])
