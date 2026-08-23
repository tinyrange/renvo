// Package app provides the setup/loop lifecycle used by small device programs.
package app

// Component is a reusable device library with one initialization phase and one
// iteration phase. Components run in slice order.
type Component interface {
	Setup()
	Loop()
}

// Setup initializes every component once.
func Setup(components []Component) {
	for i := 0; i < len(components); i++ {
		components[i].Setup()
	}
}

// Loop advances every component once.
func Loop(components []Component) {
	for i := 0; i < len(components); i++ {
		components[i].Loop()
	}
}

// Run initializes the component graph and then loops forever.
func Run(components []Component) {
	Setup(components)
	for {
		Loop(components)
	}
}
