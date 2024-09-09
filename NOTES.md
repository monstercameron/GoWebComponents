What are other GO HTML Engines?
    - https://www.gomponents.com/
    - https://github.com/a-h/templ
    - https://docs.gofiber.io/guide/templates/

What is wasm?
    - https://webassembly.org/
    - https://developer.mozilla.org/en-US/docs/WebAssembly

the original strategy was poor so I had to revise to use a tree structure to represent the DOM.
Im thinking that ill have a dom representation ont he GO side and when the HTMLElement are first mounted store their references in a map.
The idea is to link the GODom to the HTMLElement and then when state changes, I dont have to traverse the DOM to find the element I want to update, I can just update the state and the DOM will be updated.
I need to figure out how to trigger the rerender of the HTMLElement and its children.

I have been working on a "component" struct that we can write composable components that can be used to build the UI.
GO is way less flexible than JS when it comes to runtime reflection and dynamic behavior, so I have been trying to figure out how to do various things but its been a challenge.

I think I need to figure out a way to listen for js events and have that event trigger a GO Func to run.


I now have basic state setup that pipes JS events to the GO side, it works but the rerendering behavior break input fields
need to spend more time on the rerender logic such that its more fine grained. 
I need to build out a bigger app with this to flesh out POC feature set


Im gonna try my 3rd iteration fromt he DX >>> Lib Code, lets see if this yeilds good results

I was finally able to get a "fine grain" updater system going. I will clean up the code next, implement Cached fn, add documentation and then rebuild the blog portion of my website. Ill move the blog from htmx to go-html

I also need to brainstorm cool marketing names.