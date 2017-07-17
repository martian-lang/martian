#
# Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
#
# Angular controllers for martian editor main UI.
#

app = angular.module('app', ['ui.bootstrap','ngClipboard', 'googlechart'])
app.filter('shorten',  () -> (s, expand) ->
    s = s + ""
    if s.length < 71 || expand then return s
    else return s.substr(0, 30) + " ... " + s.substr(s.length - 50)
)

renderGraph = ($scope, $compile) ->
    g = new dagreD3.Digraph()
    for node in _.values($scope.nodes)
        node.label = node.name
        g.addNode(node.fqname, node)
    for node in _.values($scope.nodes)
        for edge in node.edges
            g.addEdge(null, edge.from, edge.to, {})
    (new dagreD3.Renderer()).zoom(false).run(g, d3.select("g"));
    maxX = 0.0
    d3.selectAll("g.node").each((id) ->
        d3.select(this).classed(g.node(id).type, true)
        d3.select(this).attr('ng-click', "selectNode('#{id}')")
        d3.select(this).attr('ng-class', "[node.fqname=='#{id}'?'seled':'',nodes['#{id}'].state]")
        xCoord = parseFloat(d3.select(this).attr('transform').substr(10).split(',')[0])
        if xCoord > maxX
            maxX = xCoord
    )
    maxX += 100
    if maxX < 750.0
        maxX = 750.0
    scale = 750.0 / maxX
    d3.selectAll("g#top").each((id) ->
        d3.select(this).attr('transform', 'translate(5,5) scale('+scale+')')
    )
    d3.selectAll("g.node.stage rect").each((id) ->
        d3.select(this).attr('rx', 20).attr('ry', 20))
    d3.selectAll("g.node.pipeline rect").each((id) ->
        d3.select(this).attr('rx', 0).attr('ry', 0))
    $compile(angular.element(document.querySelector('#top')).contents())($scope)

addRow = (chart, columns, name, units, stats) ->
    row = [name]
    for column in columns
        row.push {v: stats[column], f: humanize(stats[column], units)}
    chart.data.push row

addColumns = (chart, columns) ->
    chart.data = [['Stages']]
    for column in columns
        chart.data[0].push column.replace(/_/g, ' ')

humanize = (num, units) ->
    if units == 'bytes'
        s = _humanizeBytes(num)
    else if units == 'kilobytes'
        s = _humanizeBytes(num*1024)
    else if units == 'seconds'
        s = _humanizeTime(num)
    else
        s = _humanizeUnits(num, units)
    return s.trim()

_humanizeWithSuffixes = (num, suffixes, base) ->
    i = 0
    while num > base and i < suffixes.length - 1
        num = num / base
        i += 1
    return [num, suffixes[i]]

_humanizeTime = (num) ->
    [num, suffix] = _humanizeWithSuffixes(num, ['seconds', 'minutes', 'hours'], 60)
    num = num.toFixed(2)
    return num.toString()+' '+suffix

_humanizeBytes = (num) ->
    [num, suffix] = _humanizeWithSuffixes(num, ['B', 'KB', 'MB', 'GB', 'TB'], 1024)
    num = Math.round(num)
    return num.toString()+' '+suffix

_humanizeUnits = (num, units) ->
    if num >= 1000
        num = Math.round(num)
        s = ''
        l = num.toString().length
        for c, i in num.toString() by -1
            s = c + s
            if (l - i) % 3 == 0 and i > 0
                s = ',' + s
    else
        if num % 1 != 0
            num = num.toFixed(2)
        s = num.toString()
    return s+' '+units

renderChart = ($scope, columns, units) ->
    if $scope.node
        node = $scope.node
    else
        node = $scope.topnode
    pnode = $scope.pnode
    chart = {}
    chart.type = $scope.charttype
    addColumns(chart, columns)
    if pnode.type == "pipeline"
        stages = _.sortBy(pnode.forks[$scope.forki].stages, (stage) ->
            [stage.name, stage.fqname]
        )
        for stage in stages
            name = $scope.pnodes[stage.fqname].name
            fork = $scope.pnodes[stage.fqname].forks[stage.forki]
            addRow(chart, columns, name, units, fork.fork_stats)
    if pnode.type == "stage"
        fork = pnode.forks[$scope.forki]
        for chunk in fork.chunks
            addRow(chart, columns, 'chunk '+chunk.index, units, chunk.chunk_stats)
        if fork.split_stats
            addRow(chart, columns, 'split', units, fork.split_stats)
        if fork.join_stats
            addRow(chart, columns, 'join', units, fork.join_stats)
    height = Math.max(chart.data.length * 20, 100)
    chart.options = {legend: 'none', height: height, chartArea: {width: '40%', height: '90%'}}
    return chart

# Main Controller.
app.controller('MartianGraphCtrl', ($scope, $compile, $http, $interval) ->
    $scope.pname = pname
    $scope.psid = psid
    $scope.admin = admin
    $scope.adminstyle = adminstyle
    $scope.release = release
    $scope.urlprefix = if adminstyle then '/admin' else '/'
    auth = ''
    for v in window.location.search.substring(1).split("&")
        [key, val] = v.split("=")
        if key == 'auth'
            auth = '?' + v
            break

    $http.get("/api/get-state/#{container}/#{pname}/#{psid}#{auth}").success((state) ->
        $scope.topnode = state.nodes[0]
        $scope.nodes = _.indexBy(state.nodes, 'fqname')
        $scope.info = state.info
        renderGraph($scope, $compile)
    )

    $scope.id = null
    $scope.forki = 0
    $scope.chunki = 0
    $scope.mdviews = { forks:{}, split:{}, join:{}, chunks:{} }
    $scope.expand = { node:{}, forks:{}, chunks:{} }
    $scope.mdfilters = ['profile_cpu_bin', 'profile_line_bin', 'profile_mem_bin', 'heartbeat']
    $scope.showRestart = true
    $scope.showLog = false
    $scope.perf = false

    $scope.charts = {}
    $scope.charttype = 'BarChart'
    $scope.tabs = {summary: true, time: false, cpu: false, io: false, iorate: false, memory: false, jobs: false, vdr: false}
    $scope.chartopts = {
        time: {columns: ['usertime', 'systemtime'], units: 'seconds'},
        cpu: {columns: ['core_hours']},
        memory: {columns: ['maxrss'], units: 'kilobytes'},
        io: {columns: ['total_blocks', 'in_blocks', 'out_blocks']},
        iorate: {columns: ['total_blocks_rate', 'in_blocks_rate', 'out_blocks_rate']},
        jobs: {columns: ['num_jobs']},
        vdr: {columns: ['vdr_bytes'], units: 'bytes'},
    }

    # Only admin pages get auto-refresh.
    if admin
        $scope.stopRefresh = $interval(() ->
            $scope.refresh()
        , 30000)

    $scope.$watch('perf', () ->
        if $scope.perf
            $http.get("/api/get-perf/#{container}/#{pname}/#{psid}#{auth}").success((state) ->
                $scope.pnodes = _.indexBy(state.nodes, 'fqname')
                $scope.pnode = $scope.pnodes[$scope.topnode.fqname]
            )
    )

    for tab, selected of $scope.tabs
        $scope.$watch('tabs.'+tab, () ->
            $scope.getChart()
        )
    $scope.$watch('forki', () ->
        if $scope.perf
            $scope.getChart()
    )

    $scope.humanize = (name, units) ->
        fork = $scope.pnode.forks[$scope.forki]
        return humanize(fork.fork_stats[name], units)

    $scope.humanizeFromNode = (name, units) ->
        node = $scope.pnode
        return humanize(node[name], units)

    $scope.getActiveTab = () ->
        for tab, selected of $scope.tabs
            if selected
                return tab

    $scope.getChart = () ->
        active = $scope.getActiveTab()
        if $scope.chartopts[active]
            columns = $scope.chartopts[active].columns
            units = if $scope.chartopts[active].units then $scope.chartopts[active].units else ''
            $scope.charts[$scope.forki] = renderChart($scope, columns, units)

    $scope.setChartType = (charttype) ->
        $scope.charttype = charttype
        $scope.getChart()

    $scope.copyToClipboard = () ->
        return ''

    $scope.selectNode = (id) ->
        $scope.id = id
        $scope.node = $scope.nodes[id]
        $scope.forki = 0
        $scope.chunki = 0
        $scope.mdviews = { forks:{}, split:{}, join:{}, chunks:{} }
        $scope.expand = { node:{}, forks:{}, chunks:{} }
        if $scope.perf
            $scope.pnode = $scope.pnodes[id]
            $scope.getChart()

    $scope.restart = () ->
        $scope.showRestart = false
        $http.post("/api/restart/#{container}/#{pname}/#{psid}#{auth}").success((data) ->
            $scope.stopRefresh = $interval(() ->
                $scope.refresh()
            , 3000)
        ).error(() ->
            $scope.showRestart = true
            console.log('Server responded with an error for /api/restart, so stopping auto-refresh.')
            $interval.cancel($scope.stopRefresh)
            alert('mrp is no longer running.\n\nPlease run mrp again with the --noexit option to continue running the pipeline.')
        )

    $scope.expandString = (view, index, name) ->
        if !$scope.expand[view][index]?
            $scope.expand[view][index] = {}
        $scope.expand[view][index][name] = true

    $scope.selectMetadata = (view, index, name, path) ->
        $http.post("/api/get-metadata/#{container}/#{pname}/#{psid}#{auth}", { path:path, name:name }, { transformResponse: (d) -> d }).success((metadata) ->
            $scope.mdviews[view][index] = metadata
        )

    $scope.filterMetadata = (name) ->
        found = _.find($scope.mdfilters, (md) ->
            md == name
        )
        return !found

    $scope.refresh = () ->
        $http.get("/api/get-state/#{container}/#{pname}/#{psid}#{auth}").success((state) ->
            $scope.nodes = _.indexBy(state.nodes, 'fqname')
            if $scope.id then $scope.node = $scope.nodes[$scope.id]
            $scope.info = state.info
            $scope.showRestart = true
        ).error(() ->
            console.log('Server responded with an error for /api/get-state, so stopping auto-refresh.')
            $interval.cancel($scope.stopRefresh)
        )
)
