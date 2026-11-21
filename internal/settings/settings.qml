import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

// Settings window for Mouse Without Borders. It reads the profiles from the
// JSON file named by the second-to-last argument and, on Save, writes the
// result to the JSON file named by the last one. Cancel writes nothing.
ApplicationWindow {
    id: win
    visible: true
    width: 800
    height: 620
    minimumWidth: 720
    minimumHeight: 540
    title: "Mouse Without Borders - Settings"

    property var profiles: []
    property int rev: 0
    property int cur: 0
    property string active: ""
    property var deleted: []
    property string errorText: ""
    readonly property var args: Qt.application.arguments
    readonly property string inPath: args[args.length - 2]
    readonly property string outPath: args[args.length - 1]
    readonly property var current: (rev >= 0 && cur >= 0 && cur < profiles.length) ? profiles[cur] : null
    readonly property var nameRE: /^[A-Za-z0-9][A-Za-z0-9 _.-]{0,39}$/

    function touch() { rev++ }
    function setField(key, value) {
        if (!current) return
        profiles[cur][key] = value
        profiles[cur].dirty = true
        touch()
    }
    function blank(name) {
        return { name: name, isDefault: false, isNew: true, dirty: true, copyFrom: "",
                 host: "", key: "", machine: "", port: "15100", remoteWidth: "1920", remoteHeight: "1080",
                 edge: "left", clipboard: true, fileTransfer: true, captureBackend: "auto",
                 accel: "2.0", inbound: "1.0" }
    }
    function nameError(name) {
        if (name === "Default") return "Default is reserved"
        if (!nameRE.test(name) || name.charAt(name.length - 1) === " ")
            return "Use letters, digits, space, _ . - (max 40)"
        for (var i = 0; i < profiles.length; i++)
            if (profiles[i].name.toLowerCase() === name.toLowerCase()) return "That name is already used"
        return ""
    }
    function addProfile(name, copy) {
        var p = blank(name)
        if (copy && current) {
            for (var k in current) p[k] = current[k]
            p.name = name
            p.isDefault = false
            p.isNew = true
            p.dirty = true
            p.copyFrom = current.isNew ? "" : current.name
        }
        profiles.push(p)
        cur = profiles.length - 1
        touch()
    }
    function removeCurrent() {
        if (!current || current.isDefault) return
        var gone = profiles[cur]
        if (!gone.isNew) deleted.push(gone.name)
        profiles.splice(cur, 1)
        if (active === gone.name) active = "Default"
        cur = Math.max(0, cur - 1)
        touch()
    }
    function validate() {
        for (var i = 0; i < profiles.length; i++) {
            var p = profiles[i]
            var isActive = p.name === active
            if (isActive && (p.host.trim() === "" || p.key === ""))
                return "Profile \"" + p.name + "\" is in use, so it needs a host and a key."
            var port = parseInt(p.port), w = parseInt(p.remoteWidth), h = parseInt(p.remoteHeight)
            if (p.port !== "" && !(port >= 1 && port <= 65535)) return p.name + ": port must be 1-65535"
            if (p.remoteWidth !== "" && !(w > 0)) return p.name + ": remote width must be positive"
            if (p.remoteHeight !== "" && !(h > 0)) return p.name + ": remote height must be positive"
            if (p.accel !== "" && !(parseFloat(p.accel) > 0)) return p.name + ": cursor speed must be above 0"
            if (p.inbound !== "" && !(parseFloat(p.inbound) > 0)) return p.name + ": inbound speed must be above 0"
        }
        return ""
    }
    function save() {
        var err = validate()
        if (err !== "") { errorText = err; return }
        var out = { saved: true, active: active, deleted: deleted, profiles: profiles }
        var w = new XMLHttpRequest()
        w.open("PUT", "file://" + outPath)
        w.onreadystatechange = function() { if (w.readyState === 4) Qt.quit() }
        w.send(JSON.stringify(out))
    }

    Component.onCompleted: {
        var r = new XMLHttpRequest()
        r.open("GET", "file://" + inPath)
        r.onreadystatechange = function() {
            if (r.readyState !== 4) return
            var d = JSON.parse(r.responseText)
            active = d.active
            for (var i = 0; i < d.profiles.length; i++) {
                d.profiles[i].isNew = false
                d.profiles[i].dirty = false
                d.profiles[i].copyFrom = ""
                if (d.profiles[i].name === d.active) cur = i
            }
            profiles = d.profiles
            touch()
        }
        r.send()
    }

    Dialog {
        id: nameDialog
        property bool copy: false
        title: copy ? "Duplicate profile" : "New profile"
        modal: true
        anchors.centerIn: parent
        standardButtons: Dialog.Ok | Dialog.Cancel
        onOpened: { nameField.text = ""; nameField.forceActiveFocus() }
        onAccepted: win.addProfile(nameField.text.trim(), copy)
        ColumnLayout {
            spacing: 8
            Label { text: nameDialog.copy ? "Name for the copy of \"" + (win.current ? win.current.name : "") + "\":" : "Profile name:" }
            TextField { id: nameField; Layout.preferredWidth: 280; maximumLength: 40 }
            Label {
                text: nameField.text === "" ? "" : win.nameError(nameField.text.trim())
                color: "#da4453"
                visible: text !== ""
            }
        }
        Binding {
            target: nameDialog.standardButton(Dialog.Ok)
            property: "enabled"
            value: nameField.text.trim() !== "" && win.nameError(nameField.text.trim()) === ""
        }
    }

    RowLayout {
        anchors.fill: parent
        anchors.margins: 12
        spacing: 12

        // ---- Profile list ----
        ColumnLayout {
            Layout.preferredWidth: 220
            Layout.fillHeight: true
            Label { text: "Profiles"; font.bold: true }
            Frame {
                Layout.fillWidth: true
                Layout.fillHeight: true
                padding: 0
                ListView {
                    id: list
                    anchors.fill: parent
                    clip: true
                    model: win.rev >= 0 ? win.profiles.length : 0
                    currentIndex: win.cur
                    delegate: ItemDelegate {
                        width: list.width
                        highlighted: index === win.cur
                        readonly property var p: win.rev >= 0 ? win.profiles[index] : null
                        text: p ? p.name : ""
                        contentItem: RowLayout {
                            Label { text: parent.parent.text; Layout.fillWidth: true; elide: Text.ElideRight }
                            Label {
                                visible: parent.parent.p && parent.parent.p.name === win.active
                                text: "in use"
                                color: parent.parent.highlighted ? "white" : "#27ae60"
                                font.bold: true
                            }
                        }
                        onClicked: win.cur = index
                    }
                }
            }
            RowLayout {
                Layout.fillWidth: true
                Button { text: "New"; Layout.fillWidth: true; onClicked: { nameDialog.copy = false; nameDialog.open() } }
                Button { text: "Duplicate"; Layout.fillWidth: true; enabled: win.current !== null; onClicked: { nameDialog.copy = true; nameDialog.open() } }
            }
            Button {
                text: "Delete"
                Layout.fillWidth: true
                enabled: win.current !== null && !win.current.isDefault
                onClicked: win.removeCurrent()
            }
        }

        // ---- Editor ----
        ColumnLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 8

            RowLayout {
                Layout.fillWidth: true
                Label {
                    text: win.current ? win.current.name : ""
                    font.pointSize: 14
                    font.bold: true
                    Layout.fillWidth: true
                }
                Button {
                    text: (win.current && win.current.name === win.active) ? "In use" : "Use this profile"
                    enabled: win.current !== null && win.current.name !== win.active
                    highlighted: enabled
                    onClicked: { win.active = win.current.name; win.touch() }
                }
            }

            ScrollView {
                id: scroll
                Layout.fillWidth: true
                Layout.fillHeight: true
                contentWidth: availableWidth
                clip: true
                enabled: win.current !== null

                GridLayout {
                    width: scroll.availableWidth
                    columns: 2
                    columnSpacing: 12
                    rowSpacing: 8

                    Label { text: "Windows host" }
                    TextField {
                        Layout.fillWidth: true
                        placeholderText: "IP address or name of the Windows PC"
                        text: win.current ? win.current.host : ""
                        onTextEdited: win.setField("host", text)
                    }

                    Label { text: "Security key" }
                    RowLayout {
                        Layout.fillWidth: true
                        TextField {
                            id: keyField
                            Layout.fillWidth: true
                            echoMode: showKey.checked ? TextInput.Normal : TextInput.Password
                            placeholderText: "Key shown in PowerToys Mouse Without Borders"
                            text: win.current ? win.current.key : ""
                            onTextEdited: win.setField("key", text)
                        }
                        CheckBox { id: showKey; text: "Show" }
                    }

                    Label { text: "This computer's name" }
                    TextField {
                        Layout.fillWidth: true
                        placeholderText: "Defaults to the hostname (max 15 characters)"
                        maximumLength: 15
                        text: win.current ? win.current.machine : ""
                        onTextEdited: win.setField("machine", text)
                    }

                    Label { text: "Switch at screen edge" }
                    ComboBox {
                        model: ["left", "right"]
                        currentIndex: win.current && win.current.edge === "right" ? 1 : 0
                        onActivated: win.setField("edge", model[currentIndex])
                    }

                    Label { text: "Windows screen size" }
                    RowLayout {
                        TextField {
                            Layout.preferredWidth: 90
                            validator: IntValidator { bottom: 1; top: 20000 }
                            text: win.current ? win.current.remoteWidth : ""
                            onTextEdited: win.setField("remoteWidth", text)
                        }
                        Label { text: "x" }
                        TextField {
                            Layout.preferredWidth: 90
                            validator: IntValidator { bottom: 1; top: 20000 }
                            text: win.current ? win.current.remoteHeight : ""
                            onTextEdited: win.setField("remoteHeight", text)
                        }
                        Label { text: "pixels" }
                    }

                    Label { text: "Port" }
                    TextField {
                        Layout.preferredWidth: 90
                        validator: IntValidator { bottom: 1; top: 65535 }
                        text: win.current ? win.current.port : ""
                        onTextEdited: win.setField("port", text)
                    }

                    Label { text: "Cursor speed on Windows" }
                    TextField {
                        Layout.preferredWidth: 90
                        validator: DoubleValidator { bottom: 0.1; top: 20; decimals: 2; notation: DoubleValidator.StandardNotation }
                        text: win.current ? win.current.accel : ""
                        onTextEdited: win.setField("accel", text)
                    }

                    Label { text: "Cursor speed on Linux" }
                    TextField {
                        Layout.preferredWidth: 90
                        validator: DoubleValidator { bottom: 0.1; top: 20; decimals: 2; notation: DoubleValidator.StandardNotation }
                        text: win.current ? win.current.inbound : ""
                        onTextEdited: win.setField("inbound", text)
                    }

                    Label { text: "Input capture" }
                    ComboBox {
                        model: ["auto", "portal", "x11"]
                        currentIndex: win.current ? Math.max(0, model.indexOf(win.current.captureBackend)) : 0
                        onActivated: win.setField("captureBackend", model[currentIndex])
                    }

                    Item { width: 1; height: 1 }
                    CheckBox {
                        text: "Share clipboard"
                        checked: win.current ? win.current.clipboard : true
                        onToggled: win.setField("clipboard", checked)
                    }
                    Item { width: 1; height: 1 }
                    CheckBox {
                        text: "Transfer files"
                        checked: win.current ? win.current.fileTransfer : true
                        onToggled: win.setField("fileTransfer", checked)
                    }
                }
            }

            Label {
                Layout.fillWidth: true
                visible: win.errorText !== ""
                text: win.errorText
                color: "#da4453"
                wrapMode: Text.WordWrap
            }
            Label {
                Layout.fillWidth: true
                text: "Saving restarts Mouse Without Borders if the profile in use changed."
                opacity: 0.7
                wrapMode: Text.WordWrap
            }
            RowLayout {
                Layout.fillWidth: true
                Item { Layout.fillWidth: true }
                Button { text: "Cancel"; onClicked: Qt.quit() }
                Button { text: "Save"; highlighted: true; onClicked: win.save() }
            }
        }
    }
}
